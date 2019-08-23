package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
)

type SecretsManager interface {
	Login(ctx context.Context) errors.Error
	Get(ctx context.Context, key string) (*secrets.Secret, errors.Error)
}

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
