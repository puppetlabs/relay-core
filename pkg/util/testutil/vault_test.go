package testutil_test

import (
	"path"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestVaultIdentity(t *testing.T) {
	testutil.WithVault(t, func(cfg *testutil.Vault) {
		// Write some secrets.
		secrets := []struct {
			TenantID string
			Name     string
			Value    string
			Readable bool
		}{
			{
				TenantID: "nope",
				Name:     "baz",
				Value:    "hello",
				Readable: false,
			},
			{
				TenantID: "bar",
				Name:     "yes",
				Value:    "woohoo",
				Readable: true,
			},
		}
		for _, secret := range secrets {
			cfg.SetSecret(t, secret.TenantID, secret.Name, secret.Value)
		}

		// Issue a token that can access the readable secrets.
		claims := &authenticate.Claims{
			Claims: &jwt.Claims{
				Subject:  "foo",
				Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
				IssuedAt: jwt.NewNumericDate(time.Now()),
			},
			RelayTenantID: "bar",
		}

		tok, err := jwt.Signed(cfg.JWTSigner).Claims(claims).CompactSerialize()
		require.NoError(t, err)

		tc, err := api.NewClient(&api.Config{Address: cfg.Address})
		require.NoError(t, err)

		tc.ClearToken()

		auth, err := tc.Logical().Write(path.Join(cfg.JWTAuthPath, "login"), map[string]interface{}{
			"jwt":  string(tok),
			"role": cfg.JWTAuthRole,
		})
		require.NoError(t, err)

		tc.SetToken(auth.Auth.ClientToken)

		for _, secret := range secrets {
			t.Run(path.Join(secret.TenantID, secret.Name), func(t *testing.T) {
				out, err := tc.Logical().Read(path.Join(cfg.SecretsPath, "data/workflows", secret.TenantID, secret.Name))
				if secret.Readable {
					require.NoError(t, err)
					require.Equal(t, secret.Value, out.Data["data"].(map[string]interface{})["value"])
				} else {
					require.Error(t, err)
				}
			})
		}
	})
}
