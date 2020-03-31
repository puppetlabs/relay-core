package config

import (
	"github.com/puppetlabs/horsehead/v2/logging"
)

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
	// VaultToken is an optional token to use for authenticating with the
	// vaule server or agent.
	VaultToken string
	// VaultAuthMountPath is the path to use when authentication to Vault using
	// the service token. Defaults to "auth/kubernetes" if empty.
	VaultAuthMountPath string
	// ScopedSecretsPath is the store to use inside secrets backends. Added to the path
	// segment when crafting the path to a secret.
	ScopedSecretsPath string
	// K8sServiceAccountTokenPath is the path to the Service Account token. This
	// defaults to the standard kubernetes location that is set when a pod is run.
	K8sServiceAccountTokenPath string
	// K8sMasterURL is the url to the kubernetes master. If this, and the KubeconfigPath
	// are empty, then an InClusterConfig is attempted instead.
	K8sMasterURL string
	// KubeconfigPath is a path to a kubectl config file to use for authentication.
	KubeconfigPath string
	// DevelopmentPreConfigPath is a path to a configuration file that will be used
	// strictly for development. If this is set, then it's assumed in-memory managers
	// should be used and no kubernetes configuration will be set.
	DevelopmentPreConfigPath string
	// Namespace is the namespace to run the metadata-api as.
	//
	// TODO we might be able to derive a default value from this by using the client-go
	// tooling and then allow the flag to override if we want to run workflows in a
	// separate namespace to keep the metadata-api alive longer for performance reasons.
	Namespace string
	// Logger is the logger to use in all components that take this configuration.
	Logger logging.Logger
}

// WorkflowControllerConfig is the configuration object used to
// configure the Workflow controller.
type WorkflowControllerConfig struct {
	Namespace                         string
	VaultAddr                         string
	VaultToken                        string
	MetadataServiceImage              string
	MetadataServiceImagePullSecret    string
	MetadataServiceVaultAddr          string
	MetadataServiceVaultAuthMountPath string
	MetadataServiceCheckEnabled       bool
	WhenConditionsImage               string
	MaxConcurrentReconciles           int
}

// K8sClusterProvisionerConfig is the configuration object to used
// to configure the Kubernetes provisioner task.
type K8sClusterProvisionerConfig struct {
	WorkDir string
}
