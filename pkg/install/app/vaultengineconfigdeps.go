package app

import (
	"context"
	"strings"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
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

type VaultEngineConfigDeps struct {
	Core                *obj.Core
	ConfigJob           *batchv1obj.Job
	JWTSigningKeySecret *corev1obj.Secret
	OwnerConfigMap      *corev1obj.ConfigMap
	Role                *rbacv1obj.Role
	RoleBinding         *rbacv1obj.RoleBinding
	ServiceAccount      *corev1obj.ServiceAccount
	Labels              map[string]string
}

func (vd *VaultEngineConfigDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(vd.Core.Key, "owner"))

	key := helper.SuffixObjectKey(vd.Core.Key, vaultInitializationIdentifier)

	vd.ConfigJob = batchv1obj.NewJob(key)

	vd.Role = rbacv1obj.NewRole(key)
	vd.RoleBinding = rbacv1obj.NewRoleBinding(key)
	vd.ServiceAccount = corev1obj.NewServiceAccount(key)

	loaders := lifecycle.Loaders{
		vd.OwnerConfigMap,
		vd.ConfigJob,
		vd.JWTSigningKeySecret,
		vd.Role,
		vd.RoleBinding,
		vd.ServiceAccount,
	}

	if _, err := loaders.Load(ctx, cl); err != nil {
		return false, err
	}

	return true, nil
}

func (vd *VaultEngineConfigDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(vd.OwnerConfigMap.Object, owner)
}

func (vd *VaultEngineConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		vd.ConfigJob,
		vd.Role,
		vd.RoleBinding,
		vd.ServiceAccount,
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

func (vd *VaultEngineConfigDeps) Configure(ctx context.Context) error {
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
		vd.Role,
		vd.RoleBinding,
		vd.ServiceAccount,
	}

	for _, laf := range lafs {
		for label, value := range vd.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	ConfigureVaultConfigRole(vd.Role)
	ConfigureRoleBinding(vd.ServiceAccount, vd.RoleBinding)

	ConfigureVaultConfigJob(
		vd.Core.Key,
		vd.Core.Object.Spec.LogService,
		vd.Core.Object.Spec.MetadataAPI,
		vd.Core.Object.Spec.Operator,
		vd.Core.Object.Spec.Vault,
		vd.ConfigJob, vd.JWTSigningKeySecret, vd.ServiceAccount)

	return nil
}

func NewVaultSystemConfigDeps(c *obj.Core, jwt *corev1obj.Secret) *VaultEngineConfigDeps {
	vd := &VaultEngineConfigDeps{
		Core:                c,
		JWTSigningKeySecret: jwt,
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppManagedByLabel:  "relay-installer",
		},
	}

	return vd
}

func ConfigureVaultConfigRole(r *rbacv1obj.Role) {
	r.Object.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"serviceaccounts"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
	}
}

func ConfigureVaultConfigJob(
	coreKey types.NamespacedName,
	logServiceConfig *v1alpha1.LogServiceConfig,
	metadataAPIConfig v1alpha1.MetadataAPIConfig,
	operatorConfig v1alpha1.OperatorConfig,
	vaultConfig v1alpha1.VaultConfig,
	job *batchv1obj.Job, jwt *corev1obj.Secret, sa *corev1obj.ServiceAccount) {

	// TODO Determine best approach to handling slight variations in configuration data
	authPath := strings.Split(metadataAPIConfig.VaultAuthPath, "/")

	env := []corev1.EnvVar{
		{Name: metadataAPIVaultAgentRoleEnvVar, Value: metadataAPIConfig.VaultAgentRole},
		{Name: operatorVaultAgentRoleEnvVar, Value: operatorConfig.VaultAgentRole},
		{Name: tenantPathEnvVar, Value: vaultConfig.Engine.TenantPath},
		{Name: transitKeyEnvVar, Value: vaultConfig.Engine.TransitKey},
		{Name: transitPathEnvVar, Value: vaultConfig.Engine.TransitPath},
		{Name: vaultAddrEnvVar, Value: vaultConfig.Server.Address},
		{Name: vaultJWTAuthPathEnvVar, Value: metadataAPIConfig.VaultAuthPath},
		{Name: vaultJWTMountEnvVar, Value: authPath[len(authPath)-1]},
		{Name: vaultNameEnvVar, Value: vaultIdentifier},
		{Name: vaultNamespaceEnvVar, Value: coreKey.Namespace},
		{Name: vaultServiceAccountEnvVar, Value: vaultConfig.Engine.AuthDelegatorServiceAccountName},
	}

	if logServiceConfig != nil {
		env = append(env,
			corev1.EnvVar{Name: logServicePathEnvVar, Value: vaultConfig.Engine.LogServicePath},
			corev1.EnvVar{Name: logServiceVaultAgentRoleEnvVar, Value: logServiceConfig.VaultAgentRole})
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
			Image:           vaultConfig.Engine.VaultInitializationImage,
			ImagePullPolicy: vaultConfig.Engine.VaultInitializationImagePullPolicy,
			Env:             env,
		},
	}

	job.Object.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	job.Object.Spec.Template.Spec.ServiceAccountName = sa.Key.Name
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
