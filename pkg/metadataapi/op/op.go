package op

import (
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

// ManagerFactory provides data access managers for various external services
// where data resides.
type ManagerFactory interface {
	SecretsManager() (SecretsManager, errors.Error)
}

// DefaultManagerFactory is the default ManagerFactory implementation. It is very opinionated
// within the context of nebula and the workflow system.
type DefaultManagerFactory struct {
	cfg *config.MetadataServerConfig
}

func (m DefaultManagerFactory) SecretsManager() (SecretsManager, errors.Error) {
	return NewSecretsManager(m.cfg)
}

// NewManagers creates and returns a new DefaultManagerFactory
func NewDefaultManagerFactory(cfg *config.MetadataServerConfig) DefaultManagerFactory {
	return DefaultManagerFactory{cfg: cfg}
}
