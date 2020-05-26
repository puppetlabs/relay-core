package authenticate_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/inconshreveable/log15"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/horsehead/v2/scheduler"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
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

func TestVaultResolverEphemeralPorts(t *testing.T) {
	// This test checks whether the Vault resolver uses all available ports when
	// receiving tens of thousands of requests quickly on the same TCP
	// connection tuple (<server-addr>, <server-port>, <client-addr>, X) where X
	// is the chosen ephemeral port for the outbound connection.

	// This time is related to the number of requests being issued to avoid the
	// case where connections stuck in TIME_WAIT actually start to free up
	// during the test.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// TODO(NF): I observe different behavior on loopback vs. other link types
	// (including dummy link). Figure out why this is.
	testutil.WithVaultServerOnAddress(t, "127.0.0.1:0", func(addr, token string) {
		resolver := authenticate.NewStubConfigVaultResolver(addr, "auth/jwt-test")

		logging.SetLevel(log15.LvlInfo)
		defer logging.SetLevel(log15.LvlDebug)

		// You can't exit (t.FailNow, etc.) from inside a Goroutine so we use
		// one of our schedulers to collect errors for us instead.
		var pool scheduler.StartedLifecycle

		var num uint32
		proc := scheduler.SchedulableFunc(func(ctx context.Context, er scheduler.ErrorReporter) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if pn := atomic.AddUint32(&num, 1); pn > math.MaxUint16 {
					pool.Close()
					return
				} else if pn%10000 == 0 {
					t.Logf("issuing request #%d", pn)
				}

				// We don't actually care about whether the server accepts our
				// credentials, we just want to bombard it with requests.
				_, err := resolver.Resolve(ctx, authenticate.NewAuthentication(), authenticate.Raw("test"))

				// Expect a response error. Anything else (e.g. what we're
				// mainly looking for, EADDRNOTAVAIL), will come back as a
				// different type.
				if rerr, ok := err.(*vaultapi.ResponseError); !ok || rerr.StatusCode >= 500 {
					er.Put(err)
				}
			}
		})

		concurrency := 8
		if runtime.NumCPU() > concurrency {
			concurrency = runtime.NumCPU()
		}

		pool = scheduler.NewScheduler(scheduler.NManySchedulable(concurrency, proc)).
			WithErrorBehavior(scheduler.ErrorBehaviorTerminate).
			Start(scheduler.LifecycleStartOptions{})

		// Wait up to the available context for the testing to finish.
		cerr := scheduler.WaitContext(ctx, pool)

		// Close the pool (if not already closed) and wait for everything to
		// settle.
		pool.Close()
		<-pool.Done()

		// We should have no errors regardless.
		assert.Len(t, pool.Errs(), 0)

		// We only check the actual requests issued if we completed in the
		// allotted time.
		if cerr != nil {
			t.Skip("insufficient resources")
		}

		assert.True(t, num > math.MaxUint16, "only %d requests issued", num)
	})
}
