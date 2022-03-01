package app

import (
	"context"
	"fmt"
	"path"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vaultAgentConfigDirPath     = "/var/run/vault/config"
	vaultAgentConfigFileName    = "agent.hcl"
	vaultAgentConfigVolumeName  = "vault-agent-config"
	vaultAgentSATokenPath       = "/var/run/secrets/kubernetes.io/serviceaccount@vault"
	vaultAgentSATokenVolumeName = "vault-agent-sa-token"
)

type VaultAgentDeps struct {
	Core           *obj.Core
	ConfigMap      *corev1obj.ConfigMap
	ServiceAccount *corev1obj.ServiceAccount
	TokenSecret    *corev1obj.Secret
	OwnerConfigMap *corev1obj.ConfigMap
	Role           string
}

func (vd *VaultAgentDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	key := helper.SuffixObjectKey(vd.Core.Key, vd.Role)

	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "vault-agent-owner"))

	vd.ConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "vault-agent"))
	vd.ServiceAccount = corev1obj.NewServiceAccount(helper.SuffixObjectKey(key, "vault-agent"))
	vd.TokenSecret = corev1obj.NewSecret(helper.SuffixObjectKey(key, "vault-agent-token"))

	ok, err := lifecycle.Loaders{
		vd.OwnerConfigMap,
		vd.ConfigMap,
		vd.ServiceAccount,
		vd.TokenSecret,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (vd *VaultAgentDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(vd.OwnerConfigMap.Object, owner)
}

func (vd *VaultAgentDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		vd.ConfigMap,
		vd.ServiceAccount,
		vd.TokenSecret,
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

func (vd *VaultAgentDeps) Configure(ctx context.Context) error {
	if err := DependencyManager.SetDependencyOf(
		vd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: vd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	ConfigureVaultAgentTokenSecret(vd, vd.TokenSecret)
	ConfigureVaultAgentConfigMap(vd.Core, vd.Role, vd.ConfigMap)

	return nil
}

func (vd *VaultAgentDeps) DeploymentVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: vaultAgentSATokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: vd.TokenSecret.Key.Name,
				},
			},
		},
		{
			Name: vaultAgentConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: vd.ConfigMap.Key.Name,
					},
				},
			},
		},
	}
}

func (vd *VaultAgentDeps) SidecarContainer() corev1.Container {
	conf := vd.Core.Object.Spec.Vault.Sidecar

	c := corev1.Container{
		Name:            "vault-agent",
		Image:           conf.Image,
		ImagePullPolicy: conf.ImagePullPolicy,
		Resources:       conf.Resources,
		Command: []string{
			"vault",
			"agent",
			fmt.Sprintf("-config=%s",
				path.Join(vaultAgentConfigDirPath, vaultAgentConfigFileName)),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      vaultAgentSATokenVolumeName,
				ReadOnly:  true,
				MountPath: vaultAgentSATokenPath,
			},
			{
				Name:      vaultAgentConfigVolumeName,
				ReadOnly:  true,
				MountPath: vaultAgentConfigDirPath,
			},
		},
	}

	return c
}

func NewVaultAgentDepsForRole(role string, c *obj.Core) *VaultAgentDeps {
	return &VaultAgentDeps{
		Role: role,
		Core: c,
	}
}
