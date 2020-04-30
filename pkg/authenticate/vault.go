package authenticate

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"gopkg.in/square/go-jose.v2/jwt"
)

type VaultTransitIntermediary struct {
	client     *vaultapi.Client
	path       string
	key        string
	ciphertext string
}

var _ Intermediary = &VaultTransitIntermediary{}

func (vti *VaultTransitIntermediary) Next(ctx context.Context, state *Authentication) (Raw, error) {
	secret, err := vti.client.Logical().Write(vaultTransitDecryptPath(vti.path, vti.key), map[string]interface{}{
		"ciphertext": vti.ciphertext,
	})
	if err != nil {
		return nil, err
	} else if secret == nil {
		return nil, ErrNotFound
	}

	encoded, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, ErrNotFound
	}

	plaintext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return Raw(plaintext), nil
}

func NewVaultTransitIntermediary(client *vaultapi.Client, path, key, ciphertext string) *VaultTransitIntermediary {
	return &VaultTransitIntermediary{
		client:     client,
		path:       path,
		key:        key,
		ciphertext: ciphertext,
	}
}

func ChainVaultTransitIntermediary(client *vaultapi.Client, path, key string) ChainIntermediaryFunc {
	return func(ctx context.Context, raw Raw) (Intermediary, error) {
		return NewVaultTransitIntermediary(client, path, key, string(raw)), nil
	}
}

func vaultTransitDecryptPath(root, key string) string {
	return path.Join(root, "decrypt", key)
}

type VaultTransitWrapper struct {
	client *vaultapi.Client
	path   string
	key    string
}

var _ Wrapper = &VaultTransitWrapper{}

func (vtw *VaultTransitWrapper) Wrap(ctx context.Context, raw Raw) (Raw, error) {
	encoded := base64.StdEncoding.EncodeToString(raw)

	secret, err := vtw.client.Logical().Write(vaultTransitEncryptPath(vtw.path, vtw.key), map[string]interface{}{
		"plaintext": encoded,
	})
	if err != nil {
		return nil, err
	} else if secret == nil {
		return nil, fmt.Errorf("authenticate: Vault failed to return key material")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("authenticate: Vault failed to return key material")
	}

	return Raw(ciphertext), nil
}

func NewVaultTransitWrapper(client *vaultapi.Client, path, key string) *VaultTransitWrapper {
	return &VaultTransitWrapper{
		client: client,
		path:   path,
		key:    key,
	}
}

func vaultTransitEncryptPath(root, key string) string {
	return path.Join(root, "encrypt", key)
}

type VaultResolverMetadata struct {
	VaultClient *vaultapi.Client
}

type VaultResolverInjector interface {
	Inject(ctx context.Context, claims *Claims, md *VaultResolverMetadata) error
}

type VaultResolverInjectorFunc func(ctx context.Context, claims *Claims, md *VaultResolverMetadata) error

func (vf VaultResolverInjectorFunc) Inject(ctx context.Context, claims *Claims, md *VaultResolverMetadata) error {
	return vf(ctx, claims, md)
}

type VaultResolver struct {
	cfg       *vaultapi.Config
	path      string
	role      string
	injectors []VaultResolverInjector
}

var _ Resolver = &VaultResolver{}

func (vr *VaultResolver) Resolve(ctx context.Context, state *Authentication, raw Raw) (*Claims, error) {
	client, err := vaultapi.NewClient(vr.cfg)
	if err != nil {
		return nil, err
	}

	// For some reason Vault insists on reading the token from the environment
	// when a client is initialized, so unset it here for good measure.
	client.ClearToken()

	data := map[string]interface{}{
		"jwt": string(raw),
	}
	if vr.role != "" {
		data["role"] = vr.role
	}

	secret, err := client.Logical().Write(path.Join(vr.path, "login"), data)
	if err != nil {
		return nil, err
	} else if secret == nil || secret.Auth == nil {
		return nil, ErrNotFound
	}

	client.SetToken(secret.Auth.ClientToken)

	// At this point the JWT has been checked by Vault so we can just emit it
	// directly to our claims without verification.
	tok, err := jwt.ParseSigned(string(raw))
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	if err := tok.UnsafeClaimsWithoutVerification(claims); err != nil {
		return nil, err
	}

	// Add our own injectors to the state (will run after validation completes).
	md := &VaultResolverMetadata{
		VaultClient: client,
	}

	for i := range vr.injectors {
		// Capture inside loop.
		injector := vr.injectors[i]

		state.AddInjector(InjectorFunc(func(ctx context.Context, claims *Claims) error {
			return injector.Inject(ctx, claims, md)
		}))
	}

	return claims, nil
}

type VaultResolverOption func(vr *VaultResolver)

func VaultResolverWithRole(role string) VaultResolverOption {
	return func(vr *VaultResolver) {
		vr.role = role
	}
}

func VaultResolverWithInjector(injector VaultResolverInjector) VaultResolverOption {
	return func(vr *VaultResolver) {
		vr.injectors = append(vr.injectors, injector)
	}
}

func NewVaultResolver(cfg *vaultapi.Config, path string, opts ...VaultResolverOption) *VaultResolver {
	vr := &VaultResolver{
		cfg:  cfg,
		path: path,
	}

	for _, opt := range opts {
		opt(vr)
	}

	return vr
}
