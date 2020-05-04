package testutil

import (
	"testing"

	jwt "github.com/hashicorp/vault-plugin-auth-jwt"
	kv "github.com/hashicorp/vault-plugin-secrets-kv"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/builtin/logical/transit"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
	"github.com/stretchr/testify/require"
)

func WithTestVaultServer(t *testing.T, fn func(addr, token string)) {
	core, _, token := vault.TestCoreUnsealedWithConfig(t, &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"kv":      kv.Factory,
			"transit": transit.Factory,
		},
		CredentialBackends: map[string]logical.Factory{
			"jwt": jwt.Factory,
		},
		EnableUI:  false,
		EnableRaw: false,
	})
	ln, addr := http.TestServer(t, core)
	defer ln.Close()

	fn(addr, token)
}

func WithTestVaultClient(t *testing.T, fn func(client *api.Client)) {
	WithTestVaultServer(t, func(addr, token string) {
		client, err := api.NewClient(api.DefaultConfig())
		require.NoError(t, err)

		require.NoError(t, client.SetAddress(addr))
		client.SetToken(token)

		fn(client)
	})
}
