package config

type InstallerControllerConfig struct {
	MaxConcurrentReconciles int
	Namespace               string
}

func NewDefaultInstallerControllerConfig() *InstallerControllerConfig {
	return &InstallerControllerConfig{
		MaxConcurrentReconciles: 1,
		Namespace:               "default",
	}
}
