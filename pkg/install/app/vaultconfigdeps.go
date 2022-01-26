package app

import (
	"context"
	"strings"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vaultAddr                     = "http://vault:8200"
	vaultIdentifier               = "vault"
	vaultInitializationIdentifier = "vault-init"

	vaultAddrEnvVar           = "RELAY_OPERATOR_VAULT_INIT_VAULT_ADDR"
	vaultJWTAuthPathEnvVar    = "RELAY_OPERATOR_VAULT_INIT_VAULT_JWT_AUTH_PATH"
	vaultJWTMountEnvVar       = "RELAY_OPERATOR_VAULT_INIT_VAULT_JWT_MOUNT"
	vaultJWTPublicKeyEnvVar   = "RELAY_OPERATOR_VAULT_INIT_VAULT_JWT_PUBLIC_KEY"
	vaultNameEnvVar           = "RELAY_OPERATOR_VAULT_INIT_VAULT_NAME"
	vaultNamespaceEnvVar      = "RELAY_OPERATOR_VAULT_INIT_VAULT_NAMESPACE"
	vaultServiceAccountEnvVar = "RELAY_OPERATOR_VAULT_INIT_VAULT_SERVICE_ACCOUNT"
	vaultTokenEnvVarName      = "RELAY_OPERATOR_VAULT_INIT_VAULT_TOKEN"
	vaultUnsealKeyEnvVarName  = "RELAY_OPERATOR_VAULT_INIT_VAULT_UNSEAL_KEY"

	logServicePathEnvVar            = "RELAY_OPERATOR_VAULT_INIT_LOG_SERVICE_PATH"
	logServiceVaultAgentRoleEnvVar  = "RELAY_OPERATOR_VAULT_INIT_LOG_SERVICE_VAULT_AGENT_ROLE"
	metadataAPIVaultAgentRoleEnvVar = "RELAY_OPERATOR_VAULT_INIT_METADATA_API_VAULT_AGENT_ROLE"
	operatorVaultAgentRoleEnvVar    = "RELAY_OPERATOR_VAULT_INIT_OPERATOR_VAULT_AGENT_ROLE"
	tenantPathEnvVar                = "RELAY_OPERATOR_VAULT_INIT_TENANT_PATH"
	transitKeyEnvVar                = "RELAY_OPERATOR_VAULT_INIT_TRANSIT_KEY"
	transitPathEnvVar               = "RELAY_OPERATOR_VAULT_INIT_TRANSIT_PATH"
)

type VaultConfigDeps struct {
	Core                *obj.Core
	ConfigJob           *batchv1obj.Job
	JWTSigningKeySecret *corev1obj.Secret
	OwnerConfigMap      *corev1obj.ConfigMap
	Labels              map[string]string
}

func (vd *VaultConfigDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := vd.Core.Load(ctx, cl); err != nil {
		return false, err
	}

	key := helper.SuffixObjectKey(vd.Core.Key, vaultInitializationIdentifier)

	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	vd.ConfigJob = batchv1obj.NewJob(key)

	loaders := lifecycle.Loaders{
		vd.OwnerConfigMap,
		vd.ConfigJob,
		vd.JWTSigningKeySecret,
	}

	if _, err := loaders.Load(ctx, cl); err != nil {
		return false, err
	}

	return true, nil
}

func (vd *VaultConfigDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(vd.OwnerConfigMap.Object, owner)
}

func (vd *VaultConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		vd.ConfigJob,
	}

	for _, obj := range objs {
		if err := vd.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (vd *VaultConfigDeps) Configure(ctx context.Context) error {
	err := DependencyManager.SetDependencyOf(
		vd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: vd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		},
	)
	if err != nil {
		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		vd.ConfigJob,
	}

	for _, laf := range lafs {
		for label, value := range vd.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	ConfigureVaultConfigJob(
		vd.Core.Key,
		vd.Core.Object.Spec.LogService,
		vd.Core.Object.Spec.MetadataAPI,
		vd.Core.Object.Spec.Operator,
		vd.Core.Object.Spec.Vault,
		vd.ConfigJob, vd.JWTSigningKeySecret)

	return nil
}

func NewVaultConfigDeps(c *obj.Core, jwt *corev1obj.Secret) *VaultConfigDeps {
	vd := &VaultConfigDeps{
		Core:                c,
		JWTSigningKeySecret: jwt,
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}

	return vd
}

func ConfigureVaultConfigJob(
	coreKey types.NamespacedName,
	logServiceConfig v1alpha1.LogServiceConfig,
	metadataAPIConfig *v1alpha1.MetadataAPIConfig,
	operatorConfig *v1alpha1.OperatorConfig,
	vaultConfig *v1alpha1.VaultConfig,
	job *batchv1obj.Job, jwt *corev1obj.Secret) {

	// TODO Determine best approach to handling slight variations in configuration data
	authPath := strings.Split(metadataAPIConfig.VaultAuthPath, "/")

	env := []corev1.EnvVar{
		{Name: logServicePathEnvVar, Value: vaultConfig.LogServicePath},
		{Name: logServiceVaultAgentRoleEnvVar, Value: logServiceConfig.VaultAgentRole},
		{Name: metadataAPIVaultAgentRoleEnvVar, Value: metadataAPIConfig.VaultAgentRole},
		{Name: operatorVaultAgentRoleEnvVar, Value: operatorConfig.VaultAgentRole},
		{Name: tenantPathEnvVar, Value: vaultConfig.TenantPath},
		{Name: transitKeyEnvVar, Value: vaultConfig.TransitKey},
		{Name: transitPathEnvVar, Value: vaultConfig.TransitPath},
		{Name: vaultAddrEnvVar, Value: vaultAddr},
		{Name: vaultJWTAuthPathEnvVar, Value: metadataAPIConfig.VaultAuthPath},
		{Name: vaultJWTMountEnvVar, Value: authPath[len(authPath)-1]},
		{Name: vaultNameEnvVar, Value: vaultIdentifier},
		{Name: vaultNamespaceEnvVar, Value: coreKey.Namespace},
		{Name: vaultServiceAccountEnvVar, Value: vaultConfig.AuthDelegatorServiceAccountName},
	}

	if jwt != nil {
		env = append(env,
			corev1.EnvVar{
				Name: vaultJWTPublicKeyEnvVar,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: defaultPublicJWTSigningKeyName,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: jwt.Key.Name,
						},
					},
				},
			},
		)
	}

	if vaultConfig.Auth != nil {
		if token, ok := VaultAuthDataEnvVar(vaultTokenEnvVarName, vaultConfig.Auth.Token); ok {
			env = append(env, token)
		}

		if unsealKey, ok := VaultAuthDataEnvVar(vaultUnsealKeyEnvVarName, vaultConfig.Auth.UnsealKey); ok {
			env = append(env, unsealKey)
		}
	}

	job.Object.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            vaultInitializationIdentifier,
			Image:           vaultConfig.VaultInitializationImage,
			ImagePullPolicy: vaultConfig.VaultInitializationImagePullPolicy,
			Env:             env,
		},
	}

	job.Object.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever

	// TODO Consider supporting separate RBAC for use with this container (or at the very least, should not be hardcoded here)
	job.Object.Spec.Template.Spec.ServiceAccountName = "relay-install-operator"
}

func VaultAuthDataEnvVar(name string, vad *v1alpha1.VaultAuthData) (corev1.EnvVar, bool) {
	if vad != nil {
		if vad.Value != "" {
			return corev1.EnvVar{
				Name:  name,
				Value: vad.Value,
			}, true
		}

		if vad.ValueFrom != nil {
			if vad.ValueFrom.SecretKeyRef != nil {
				return corev1.EnvVar{
					Name: name,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: vad.ValueFrom.SecretKeyRef,
					},
				}, true
			}

			return corev1.EnvVar{}, false
		}
	}

	return corev1.EnvVar{}, false
}
