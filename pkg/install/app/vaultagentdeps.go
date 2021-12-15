package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vaultAgentConfigDirPath = "/var/run/vault/config"
	vaultAgentSATokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount@vault"
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

func (vd *VaultAgentDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		vd.ConfigMap,
		vd.ServiceAccount,
		vd.TokenSecret,
	}

	for _, o := range os {
		if err := vd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		vd.ConfigMap,
		vd.ServiceAccount,
		vd.TokenSecret,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewVaultAgentDepsForRole(role string, c *obj.Core) *VaultAgentDeps {
	return &VaultAgentDeps{
		Role: role,
		Core: c,
	}
}

func ConfigureVaultAgentDeps(vd *VaultAgentDeps) error {
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
