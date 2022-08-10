package testutil

import (
	"net"
	"net/url"
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

func WithVaultServerOnAddress(t *testing.T, addr string, fn func(addr, token string)) {
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

	ln, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	defer ln.Close()

	addr = (&url.URL{Scheme: "http", Host: ln.Addr().String()}).String()
	http.TestServerWithListener(t, ln, addr, core)
	fn(addr, token)
}

func WithVaultServer(t *testing.T, fn func(addr, token string)) {
	WithVaultServerOnAddress(t, "127.0.0.1:0", fn)
}

func WithVaultClient(t *testing.T, fn func(client *api.Client)) {
	WithVaultServer(t, func(addr, token string) {
		client, err := api.NewClient(api.DefaultConfig())
		require.NoError(t, err)

		require.NoError(t, client.SetAddress(addr))
		client.SetToken(token)

		fn(client)
	})
}
