package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
)

// SecretsManager is responsible for accessing the backend where secrets are stored
// and retrieving values for a given key.
type SecretsManager interface {
	Login(ctx context.Context) errors.Error
	Get(ctx context.Context, key string) (*secrets.Secret, errors.Error)
}

// NewSecretsManager creates and returns a SecretsManager.
// Currently this returns the Vault implementation, but can be used to create
// alternative engines derived from options in cfg.
func NewSecretsManager(cfg *config.MetadataServerConfig) (SecretsManager, errors.Error) {
	sec, err := vault.NewVaultWithKubernetesAuth(&vault.Config{
		Addr:                       cfg.VaultAddr,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Role:                       cfg.VaultRole,
		Bucket:                     cfg.WorkflowID,
		EngineMount:                cfg.VaultEngineMount,
	})
	if err != nil {
		return nil, err
	}

	return sec, nil
}
