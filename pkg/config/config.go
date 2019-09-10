package config

import "github.com/puppetlabs/horsehead/logging"

// MetadataServerConfig is the configuration object used to configure
// the metadata http server.
type MetadataServerConfig struct {
	// BindAddr is the address and port to bind the server to
	BindAddr string
	// VaultAddr is the http address to the vault server or agent
	VaultAddr string
	// VaultRole is the role to use when logging into vault. In the context of
	// Nebula and the metadata-api, this is most-likely the kubernetes namespace
	// that the workflow is running under.
	VaultRole string
	// VaultEngineMount is the store to use inside vault. Added to the path
	// segment when crafting the path to a secret.
	VaultEngineMount string
	// VaultToken is an optional token to use for authenticating with the
	// vaule server or agent.
	VaultToken string
	// WorkflowID is the ID of the workflow to run the metadata-api under
	WorkflowID string
	// K8sServiceAccountTokenPath is the path to the Service Account token. This
	// defaults to the standard kubernetes location that is set when a pod is run.
	K8sServiceAccountTokenPath string
	// Namespace is the namespace to run the metadata-api as.
	//
	// TODO we might be able to derive a default value from this by using the client-go
	// tooling and then allow the flag to override if we want to run workflows in a
	// separate namespace to keep the metadata-api alive longer for performance reasons.
	Namespace string
	// Logger is the logger to use in all components that take this configuration.
	Logger logging.Logger
}

// SecretAuthControllerConfig is the configuration object used to
// configure the SecretAuth controller that creates a security context
// for namespaced pods to access their secret values from vault.
type SecretAuthControllerConfig struct {
	Kubeconfig                     string
	KubeMasterURL                  string
	VaultAddr                      string
	VaultToken                     string
	MetadataServiceImage           string
	MetadataServiceImagePullSecret string
	MetadataServiceVaultAddr       string
}

// K8sClusterProvisionerConfig is the configuration object to used
// to configure the Kubernetes provisioner task.
type K8sClusterProvisionerConfig struct {
	WorkDir string
}
