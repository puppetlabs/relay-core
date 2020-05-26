package authenticate

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	vaultapi "github.com/hashicorp/vault/api"
	"gopkg.in/square/go-jose.v2/jwt"
)

func VaultTransitNamespaceContext(namespace string) string {
	return fmt.Sprintf("k8s.io/namespaces/%s", namespace)
}

type VaultTransitIntermediary struct {
	client     *vaultapi.Client
	path       string
	key        string
	context    string
	ciphertext string
}

var _ Intermediary = &VaultTransitIntermediary{}

func (vti *VaultTransitIntermediary) Next(ctx context.Context, state *Authentication) (Raw, error) {
	data := map[string]interface{}{
		"ciphertext": vti.ciphertext,
	}
	if vti.context != "" {
		data["context"] = base64.StdEncoding.EncodeToString([]byte(vti.context))
	}

	secret, err := vti.client.Logical().Write(vaultTransitDecryptPath(vti.path, vti.key), data)
	if err != nil {
		return nil, err
	} else if secret == nil {
		return nil, &NotFoundError{Reason: "vault: transit: no secret data in response (bug?)"}
	}

	encoded, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, &NotFoundError{Reason: "vault: transit: plaintext missing from secret data (bug?)"}
	}

	plaintext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	return Raw(plaintext), nil
}

type VaultTransitIntermediaryOption func(vti *VaultTransitIntermediary)

func VaultTransitIntermediaryWithContext(context string) VaultTransitIntermediaryOption {
	return func(vti *VaultTransitIntermediary) {
		vti.context = context
	}
}

func NewVaultTransitIntermediary(client *vaultapi.Client, path, key, ciphertext string, opts ...VaultTransitIntermediaryOption) *VaultTransitIntermediary {
	vti := &VaultTransitIntermediary{
		client:     client,
		path:       path,
		key:        key,
		ciphertext: ciphertext,
	}

	for _, opt := range opts {
		opt(vti)
	}

	return vti
}

func ChainVaultTransitIntermediary(client *vaultapi.Client, path, key string, opts ...VaultTransitIntermediaryOption) ChainIntermediaryFunc {
	return func(ctx context.Context, raw Raw) (Intermediary, error) {
		return NewVaultTransitIntermediary(client, path, key, string(raw), opts...), nil
	}
}

func vaultTransitDecryptPath(root, key string) string {
	return path.Join(root, "decrypt", key)
}

type VaultTransitWrapper struct {
	client  *vaultapi.Client
	path    string
	key     string
	context string
}

var _ Wrapper = &VaultTransitWrapper{}

func (vtw *VaultTransitWrapper) Wrap(ctx context.Context, raw Raw) (Raw, error) {
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(raw),
	}
	if vtw.context != "" {
		data["context"] = base64.StdEncoding.EncodeToString([]byte(vtw.context))
	}

	secret, err := vtw.client.Logical().Write(vaultTransitEncryptPath(vtw.path, vtw.key), data)
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

type VaultTransitWrapperOption func(vtw *VaultTransitWrapper)

func VaultTransitWrapperWithContext(context string) VaultTransitWrapperOption {
	return func(vtw *VaultTransitWrapper) {
		vtw.context = context
	}
}

func NewVaultTransitWrapper(client *vaultapi.Client, path, key string, opts ...VaultTransitWrapperOption) *VaultTransitWrapper {
	vtw := &VaultTransitWrapper{
		client: client,
		path:   path,
		key:    key,
	}

	for _, opt := range opts {
		opt(vtw)
	}

	return vtw
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
		return nil, &NotFoundError{Reason: "vault: resolver: no authentication information in response"}
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

func NewStubConfigVaultResolver(addr, path string, opts ...VaultResolverOption) *VaultResolver {
	// This is similar to the Vault default config but doesn't look in the
	// environment for anything.
	ht := cleanhttp.DefaultPooledTransport()

	// Since we have a single target address here, we're ideally going to
	// customize the transport to allow a lot of idle connections to it. Since
	// the resolver runs for each incoming request we want to prevent defunct
	// connections from piling up in TIME_WAIT but also not have to deal with
	// the overhead of opening a new connection for every request.
	//
	// The settings I'd like to use look something like:
	//   ht.MaxIdleConns = 1024
	//   ht.MaxIdleConnsPerHost = 1024
	//   ht.MaxConnsPerHost = 1024
	// However, these actually make the test case worse (or just as bad?). It
	// seems that some of the connections don't get reused... not sure if this
	// is a client bug or misunderstanding of how the options are supposed to
	// work.
	//
	// So we'll do this the blunt way and, for now, disable keep-alives
	// altogether:
	ht.DisableKeepAlives = true
	ht.MaxIdleConnsPerHost = -1

	cfg := &vaultapi.Config{
		Address: addr,
		HttpClient: &http.Client{
			Transport: ht,
		},
		Timeout:    60 * time.Second,
		Backoff:    retryablehttp.LinearJitterBackoff,
		MaxRetries: 2,
	}

	// See comments in the Vault code for this.
	cfg.HttpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return NewVaultResolver(cfg, path, opts...)
}
