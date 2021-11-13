package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
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
	Role           obj.VaultAgentRole
}

func (vd *VaultAgentDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	key := SuffixObjectKey(vd.Core.Key, string(vd.Role))

	vd.OwnerConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "vault-agent-owner"))

	vd.ConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "vault-agent"))
	vd.ServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "vault-agent"))
	vd.TokenSecret = corev1obj.NewSecret(SuffixObjectKey(key, "vault-agent-token"))

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

func NewVaultAgentDepsForRole(role obj.VaultAgentRole, c *obj.Core) *VaultAgentDeps {
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

func ConfigureVaultAgentContainer(core *obj.Core, c *corev1.Container) {
	c.Name = "vault-agent"
	c.Image = core.Object.Spec.Vault.Sidecar.Image
	c.ImagePullPolicy = core.Object.Spec.Vault.Sidecar.ImagePullPolicy
	c.Resources = core.Object.Spec.Vault.Sidecar.Resources

	c.Command = []string{
		"vault",
		"agent",
		"-config=/var/run/vault/config/agent.hcl",
	}

	c.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "vault-agent-config",
			ReadOnly:  true,
			MountPath: vaultAgentConfigDirPath,
		},
		{
			Name:      "vault-agent-sa-token",
			ReadOnly:  true,
			MountPath: vaultAgentSATokenPath,
		},
	}
}

func ConfigureVaultAgentTokenSecret(vd *VaultAgentDeps, secret *corev1obj.Secret) {
	if secret.Object.Annotations == nil {
		secret.Object.Annotations = make(map[string]string)
	}

	secret.Object.Annotations[corev1.ServiceAccountNameKey] = vd.ServiceAccount.Key.Name
	secret.Object.Type = corev1.SecretTypeServiceAccountToken
}

func ConfigureVaultAgentConfigMap(core *obj.Core, role obj.VaultAgentRole, cm *corev1obj.ConfigMap) {
	if cm.Object.Data == nil {
		cm.Object.Data = make(map[string]string)
	}

	config := VaultAgentConfig{
		AutoAuth: &VaultAutoAuth{
			Method: &VaultAutoAuthMethod{
				Type:      "kubernetes",
				MountPath: "auth/kubernetes",
				Config: map[string]string{
					"role":       string(role),
					"token_path": "/var/run/secrets/kubernetes.io/serviceaccount@vault/token",
				},
			},
		},
		Cache: &VaultCache{
			UseAutoAuthToken: true,
		},
		Listeners: []*VaultListener{
			{
				Type:        "tcp",
				Address:     "127.0.0.1",
				TLSDisabled: true,
			},
		},
		Vault: &VaultServer{
			Address: core.Object.Spec.Vault.Sidecar.ServerAddr,
		},
	}

	b := generateVaultConfig(&config)

	cm.Object.Data["agent.hcl"] = string(b)
}
