package v1beta1

type VaultConfig struct {
	// URL is the fully-qualified location of the Vault service.
	//
	// +optional
	// +kubebuilder:default="http://localhost:8200"
	URL string `json:"url,omitempty"`

	// Token is the Vault token to use.
	//
	// +optional
	Token string `json:"token,omitempty"`

	// TokenFrom allows the Vault token to be provided by another resource.
	//
	// +optional
	TokenFrom *VaultTokenSource `json:"tokenFrom,omitempty"`
}

type VaultTokenSource struct {
	// SecretKeyRef looks up a Vault token in a Kubernetes secret.
	//
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`

	// JWT causes the metadata API to use the given role and authentication
	// backend to acquire a token with the JWT provided to each action.
	//
	// +optional
	JWT *JWTVaultTokenSource `json:"jwt,omitempty"`
}

type JWTVaultTokenSource struct {
	// Path is the base path to the JWT authentication method, like "auth/jwt".
	//
	// +optional
	// +kubebuilder:default="auth/jwt"
	Path string `json:"path,omitempty"`

	// Role is the name of the configured authentication role to use in Vault.
	Role string `json:"role"`
}

type VaultEngineScheme string

const (
	VaultEngineSchemeDefault = ""
	VaultEngineSchemeKVV2    = "kv-v2"
)

type VaultEngine struct {
	VaultConfig `json:",inline"`

	// Scheme is the mechanism to use to look up the values in a folder.
	//
	// For the default scheme, it is assumed that children are read by suffixing
	// their name to the path given.
	//
	// For the KV v2 scheme, if the given path does not begin with "metadata/",
	// it is prefixed before listing. When subsequently reading values at a
	// path, the "metadata/" prefix is changed to "data/".
	//
	// +optional
	Scheme VaultEngineScheme `json:"scheme,omitempty"`

	// Path is the base path to the engine, like "secret".
	//
	// +optional
	// +kubebuilder:default="secret"
	Path string `json:"path,omitempty"`
}

type VaultPathSelector struct {
	// Engine is the Vault engine to use. If not specified, the default path of
	// "secret" is used and the connection configuration is specified by the
	// tenant. If the tenant does not specify a Vault configuration,
	// unauthenticated requests are made over HTTP to localhost:8200.
	//
	// +optional
	Engine *VaultEngine `json:"engine,omitempty"`

	// Path is the location of the secret fields to read in Vault.
	Path string `json:"path"`
}

type VaultFieldSelector struct {
	VaultPathSelector `json:",inline"`

	// Field is the key within the path that contains the secret data.
	Field string `json:"field"`
}
