package config

type MetadataServerConfig struct {
	BindAddr  string
	VaultAddr string
}

type SecretAuthControllerConfig struct {
	Kubeconfig    string
	KubeMasterURL string
	VaultAddr     string
	VaultToken    string
}
