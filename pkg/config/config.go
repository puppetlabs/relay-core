package config

// MetadataServerConfig is the configuration object used to configure
// the metadata http server.
type MetadataServerConfig struct {
	BindAddr  string
	VaultAddr string
	Namespace string
}

// SecretAuthControllerConfig is the configuration object used to
// configure the SecretAuth controller that creates a security context
// for namespaced pods to access their secret values from vault.
type SecretAuthControllerConfig struct {
	Kubeconfig    string
	KubeMasterURL string
	VaultAddr     string
	VaultToken    string
}
