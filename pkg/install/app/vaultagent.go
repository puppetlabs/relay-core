package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureVaultAgentTokenSecret(vd *VaultAgentDeps, secret *corev1obj.Secret) {
	if secret.Object.Annotations == nil {
		secret.Object.Annotations = make(map[string]string)
	}

	secret.Object.Annotations[corev1.ServiceAccountNameKey] = vd.ServiceAccount.Key.Name
	secret.Object.Type = corev1.SecretTypeServiceAccountToken
}

func ConfigureVaultAgentConfigMap(core *obj.Core, role string, cm *corev1obj.ConfigMap) {
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
				Type:       "tcp",
				Address:    "127.0.0.1:8200",
				TLSDisable: true,
			},
		},
		Vault: &VaultServer{
			Address: core.Object.Spec.Vault.Server.Address,
		},
	}

	b := generateVaultConfig(&config)

	cm.Object.Data[vaultAgentConfigFileName] = string(b)
}
