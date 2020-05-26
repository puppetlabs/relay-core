package testutil

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/url"
	"path"
	"testing"
	"text/template"

	"github.com/google/uuid"
	jwt "github.com/hashicorp/vault-plugin-auth-jwt"
	kv "github.com/hashicorp/vault-plugin-secrets-kv"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/builtin/logical/transit"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
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

var VaultMetadataAPIPolicyTemplate = template.Must(template.
	New("vault_metadata_api_policy_template").
	Delims("${", "}").
	Parse(`
path "${ .SecretsPath }/data/workflows/{{identity.entity.aliases.${ .Accessor }.metadata.tenant_id}}/*" {
	capabilities = ["read"]
}

path "${ .SecretsPath }/metadata/connections/{{identity.entity.aliases.${ .Accessor }.metadata.domain_id}}/*" {
	capabilities = ["list"]
}

path "${ .SecretsPath }/data/connections/{{identity.entity.aliases.${ .Accessor }.metadata.domain_id}}/*" {
	capabilities = ["read"]
}
`))

type VaultMetadataAPIPolicyTemplateData struct {
	SecretsPath string
	Accessor    string
}

type Vault struct {
	Address string
	Client  *api.Client

	SecretsPath string

	TransitPath string
	TransitKey  string

	JWTSigner    jose.Signer
	JWTPublicKey crypto.PublicKey
	JWTAuthPath  string
	JWTAuthRole  string
}

func (v *Vault) SetSecret(t *testing.T, tenantID, name, value string) {
	_, err := v.Client.Logical().Write(path.Join(v.SecretsPath, "data/workflows", tenantID, name), map[string]interface{}{
		"data": map[string]interface{}{
			"value": value,
		},
	})
	require.NoError(t, err)
}

func (v *Vault) SetConnection(t *testing.T, domainID, typ, name string, attrs map[string]string) {
	id := uuid.New().String()

	for k, vi := range attrs {
		_, err := v.Client.Logical().Write(path.Join(v.SecretsPath, "data/connections", domainID, id, k), map[string]interface{}{
			"data": map[string]interface{}{
				"value": vi,
			},
		})
		require.NoError(t, err)
	}

	_, err := v.Client.Logical().Write(path.Join(v.SecretsPath, "data/connections", domainID, typ, name), map[string]interface{}{
		"data": map[string]interface{}{
			"value": id,
		},
	})
	require.NoError(t, err)
}

// WithVault creates a Vault server and client preconfigured with all of the
// various engines needed to make the controller and metadata API work together.
func WithVault(t *testing.T, fn func(cfg *Vault)) {
	WithVaultClient(t, func(client *api.Client) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: key}, &jose.SignerOptions{})
		require.NoError(t, err)

		cfg := &Vault{
			Address: client.Address(),
			Client:  client,

			SecretsPath: "kv-secrets",

			TransitPath: "transit-core",
			TransitKey:  "metadata-api",

			JWTSigner:    signer,
			JWTPublicKey: key.Public(),
			JWTAuthPath:  "auth/jwt-core",
			JWTAuthRole:  "metadata-api",
		}

		// Secrets
		require.NoError(t, client.Sys().Mount(cfg.SecretsPath, &api.MountInput{
			Type: "kv-v2",
		}))

		// Transit
		require.NoError(t, client.Sys().Mount(cfg.TransitPath, &api.MountInput{
			Type: "transit",
		}))

		_, err = client.Logical().Write(path.Join(cfg.TransitPath, "keys", cfg.TransitKey), map[string]interface{}{
			"derived": true,
		})
		require.NoError(t, err)

		// Authentication
		require.NoError(t, client.Sys().EnableAuthWithOptions(path.Base(cfg.JWTAuthPath), &api.EnableAuthOptions{
			Type: "jwt",
		}))

		mounts, err := client.Sys().ListAuth()
		require.NoError(t, err)

		var policy bytes.Buffer
		require.NoError(t, VaultMetadataAPIPolicyTemplate.Execute(&policy, VaultMetadataAPIPolicyTemplateData{
			SecretsPath: cfg.SecretsPath,
			Accessor:    mounts[path.Base(cfg.JWTAuthPath)+"/"].Accessor,
		}))

		require.NoError(t, client.Sys().PutPolicy("relay/metadata-api", policy.String()))

		_, err = client.Logical().Write(path.Join(cfg.JWTAuthPath, "role", cfg.JWTAuthRole), map[string]interface{}{
			"name":            cfg.JWTAuthRole,
			"role_type":       "jwt",
			"bound_audiences": []string{authenticate.MetadataAPIAudienceV1},
			"user_claim":      "sub",
			"token_type":      "batch",
			"token_policies":  []string{"relay/metadata-api"},
			"claim_mappings": map[string]interface{}{
				"relay.sh/domain-id": "domain_id",
				"relay.sh/tenant-id": "tenant_id",
			},
		})
		require.NoError(t, err)

		pub, err := x509.MarshalPKIXPublicKey(key.Public())
		require.NoError(t, err)

		_, err = client.Logical().Write(path.Join(cfg.JWTAuthPath, "config"), map[string]interface{}{
			"jwt_validation_pubkeys": []string{string(pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: pub,
			}))},
			"jwt_supported_algs": []string{"RS256", "RS512"},
		})
		require.NoError(t, err)

		fn(cfg)
	})
}
