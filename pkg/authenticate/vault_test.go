package authenticate_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestVaultTransitIntermediary(t *testing.T) {
	ctx := context.Background()

	testutil.WithVaultClient(t, func(vc *vaultapi.Client) {
		// Vault configuration:
		require.NoError(t, vc.Sys().Mount("transit-test", &vaultapi.MountInput{
			Type: "transit",
		}))

		_, err := vc.Logical().Write("transit-test/keys/metadata-api", map[string]interface{}{
			"derived": true,
		})
		require.NoError(t, err)

		// Encrypt the token.
		secret, err := vc.Logical().Write("transit-test/encrypt/metadata-api", map[string]interface{}{
			"plaintext": base64.StdEncoding.EncodeToString([]byte("my-auth-token")),
			"context":   base64.StdEncoding.EncodeToString([]byte("hello")),
		})
		require.NoError(t, err)

		encryptedToken, ok := secret.Data["ciphertext"].(string)
		require.True(t, ok, "ciphertext is not a string")
		require.NotEmpty(t, encryptedToken)

		im := authenticate.NewVaultTransitIntermediary(
			vc, "transit-test", "metadata-api", encryptedToken,
			authenticate.VaultTransitIntermediaryWithContext("hello"),
		)
		raw, err := im.Next(ctx, authenticate.NewAuthentication())
		require.NoError(t, err)
		require.Equal(t, authenticate.Raw("my-auth-token"), raw)
	})
}

func TestVaultTransitWrapper(t *testing.T) {
	ctx := context.Background()

	testutil.WithVaultClient(t, func(vc *vaultapi.Client) {
		// Vault configuration:
		require.NoError(t, vc.Sys().Mount("transit-test", &vaultapi.MountInput{
			Type: "transit",
		}))

		_, err := vc.Logical().Write("transit-test/keys/metadata-api", map[string]interface{}{
			"derived": true,
		})
		require.NoError(t, err)

		wrapper := authenticate.NewVaultTransitWrapper(
			vc, "transit-test", "metadata-api",
			authenticate.VaultTransitWrapperWithContext("hello"),
		)
		raw, err := wrapper.Wrap(ctx, authenticate.Raw("my-auth-token"))
		require.NoError(t, err)

		// Decrypt the token.
		secret, err := vc.Logical().Write("transit-test/decrypt/metadata-api", map[string]interface{}{
			"ciphertext": string(raw),
			"context":    base64.StdEncoding.EncodeToString([]byte("hello")),
		})
		require.NoError(t, err)

		encodedToken, ok := secret.Data["plaintext"].(string)
		require.True(t, ok)

		token, err := base64.StdEncoding.DecodeString(encodedToken)
		require.NoError(t, err)
		require.Equal(t, []byte("my-auth-token"), token)
	})
}

func TestVaultResolver(t *testing.T) {
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: key}, &jose.SignerOptions{})
	require.NoError(t, err)

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Subject:   "foo",
			Audience:  []string{"test-aud"},
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	tok, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	require.NoError(t, err)

	pub, err := x509.MarshalPKIXPublicKey(key.Public())
	require.NoError(t, err)

	testutil.WithVaultClient(t, func(vc *vaultapi.Client) {
		require.NoError(t, vc.Sys().EnableAuthWithOptions("jwt-test", &vaultapi.EnableAuthOptions{
			Type: "jwt",
		}))

		_, err := vc.Logical().Write("auth/jwt-test/role/test", map[string]interface{}{
			"name":            "test",
			"role_type":       "jwt",
			"bound_audiences": []string{"test-aud"},
			"user_claim":      "sub",
			"token_type":      "batch",
		})
		require.NoError(t, err)

		_, err = vc.Logical().Write("auth/jwt-test/config", map[string]interface{}{
			"jwt_validation_pubkeys": []string{string(pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: pub,
			}))},
			"jwt_supported_algs": []string{"RS256", "RS512"},
		})
		require.NoError(t, err)

		resolver := authenticate.NewStubConfigVaultResolver(
			vc.Address(),
			"auth/jwt-test",
			authenticate.VaultResolverWithRole("test"),
		)
		claims, err := resolver.Resolve(ctx, authenticate.NewAuthentication(), authenticate.Raw(tok))
		require.NoError(t, err)
		require.Equal(t, "foo", claims.Subject)
	})
}
