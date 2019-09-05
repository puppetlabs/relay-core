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
	OutputsManager() OutputsManager
	MetadataManager() MetadataManager
	KubernetesManager() KubernetesManager
}

// DefaultManagerFactory is the default ManagerFactory implementation. It is very opinionated
// within the context of nebula and the workflow system.
type DefaultManagerFactory struct {
	cfg *config.MetadataServerConfig
	sm  SecretsManager
	om  OutputsManager
	mm  MetadataManager
	km  KubernetesManager
}

// SecretsManager creates and returns a new SecretsManager implementation.
func (m DefaultManagerFactory) SecretsManager() SecretsManager {
	return m.sm
}

// OutputsManager creates and returns a new OutputsManager based on values in Configuration type.
func (m DefaultManagerFactory) OutputsManager() OutputsManager {
	return m.om
}

func (m DefaultManagerFactory) MetadataManager() MetadataManager {
	return m.mm
}

func (m DefaultManagerFactory) KubernetesManager() KubernetesManager {
	return m.km
}

// NewDefaultManagerFactory creates and returns a new DefaultManagerFactory
func NewDefaultManagerFactory(ctx context.Context, cfg *config.MetadataServerConfig) (*DefaultManagerFactory, errors.Error) {
	kc, err := NewKubeclientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	cfg.Kubeclient = kc

	km := NewDefaultKubernetesManager(kc)

	secretsBackend := SecretsBackendMapping["vault"]
	sm, err := SecretsBackendAdapters[secretsBackend](ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.OutputsBackend == "" {
		cfg.Logger.Warn("using in memory outputs storage; this is not recommended for production")

		cfg.OutputsBackend = "memory"
	}

	outputsBackend, ok := OutputsBackendMapping[cfg.OutputsBackend]
	if !ok {
		return nil, errors.NewOutputsBackendDoesNotExist(cfg.OutputsBackend)
	}

	om := OutputsBackendAdapters[outputsBackend](cfg)

	return &DefaultManagerFactory{
		cfg: cfg,
		sm:  sm,
		om:  om,
		km:  km,
	}, nil
}
