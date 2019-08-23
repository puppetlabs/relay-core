package op

import (
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Managers struct {
	cfg *config.MetadataServerConfig
}

func (m Managers) SecretsManager() (SecretsManager, errors.Error) {
	return NewSecretsManager(m.cfg)
}

func NewManagers(cfg *config.MetadataServerConfig) Managers {
	return Managers{cfg: cfg}
}
