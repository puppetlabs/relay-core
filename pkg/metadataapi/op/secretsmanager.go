package op

import (
	"context"

	"github.com/puppetlabs/horsehead/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
)

// SecretsManager is responsible for accessing the backend where secrets are stored
// and retrieving values for a given key.
type SecretsManager interface {
	Get(ctx context.Context, key string) (*secrets.Secret, errors.Error)
}

// NewSecretsManager creates and returns a SecretsManager.
// Currently this returns the Vault implementation, but can be used to create
// alternative engines derived from options in cfg.
func NewSecretsManager(ctx context.Context, cfg *config.MetadataServerConfig) (SecretsManager, errors.Error) {
	sm, err := vault.NewVaultWithKubernetesAuth(ctx, &vault.Config{
		Addr:                       cfg.VaultAddr,
		K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
		Token:                      cfg.VaultToken,
		Role:                       cfg.VaultRole,
		Bucket:                     cfg.WorkflowID,
		EngineMount:                cfg.VaultEngineMount,
		Logger:                     cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	return NewEncodingSecretManager(sm), nil
}

// EncodingSecretsManager wraps a SecretManager and decodes the secrets.Secret.Value using
// the default encoding/transfer library.
type EncodingSecretsManager struct {
	delegate SecretsManager
}

func (m EncodingSecretsManager) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	sec, err := m.delegate.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	decoded, derr := transfer.DecodeFromTransfer(sec.Value)
	if derr != nil {
		return nil, errors.NewSecretsValueDecodingError().WithCause(derr)
	}

	sec.Value = string(decoded)

	return sec, nil
}

func NewEncodingSecretManager(sm SecretsManager) *EncodingSecretsManager {
	return &EncodingSecretsManager{delegate: sm}
}
