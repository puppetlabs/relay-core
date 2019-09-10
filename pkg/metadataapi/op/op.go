package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

// ManagerFactory provides data access managers for various external services
// where data resides.
type ManagerFactory interface {
	SecretsManager() SecretsManager
}

// DefaultManagerFactory is the default ManagerFactory implementation. It is very opinionated
// within the context of nebula and the workflow system.
type DefaultManagerFactory struct {
	cfg *config.MetadataServerConfig
	sm  SecretsManager
}

// SecretsManager creates and returns a new SecretsManager implementation.
func (m DefaultManagerFactory) SecretsManager() SecretsManager {
	return m.sm
}

// NewDefaultManagerFactory creates and returns a new DefaultManagerFactory
func NewDefaultManagerFactory(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	sm, err := NewSecretsManager(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &DefaultManagerFactory{
		cfg: cfg,
		sm:  sm,
	}, nil
}
