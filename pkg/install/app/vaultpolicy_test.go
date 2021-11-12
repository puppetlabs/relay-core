package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVaultAgentConfig(t *testing.T) {
	var expected = `
auto_auth {

  method "kubernetes" {
    mount_path = "auth/kubernetes"
    config     = { role = "operator", token_path = "/var/run/secrets/kubernetes.io/serviceaccount@vault/token" }
  }
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address      = "127.0.0.1:8200"
  tls_disabled = true
}

vault {
  address = "unit-testing.relay.sh:8200"
}
`

	cfg := &VaultAgentConfig{
		AutoAuth: &VaultAutoAuth{
			Method: &VaultAutoAuthMethod{
				Type:      "kubernetes",
				MountPath: "auth/kubernetes",
				Config: map[string]string{
					"role":       "operator",
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
				Address:     "127.0.0.1:8200",
				TLSDisabled: true,
			},
		},
		Vault: &VaultServer{
			Address: "unit-testing.relay.sh:8200",
		},
	}

	b := generateVaultConfig(cfg)
	require.Equal(t, expected, string(b))
}
