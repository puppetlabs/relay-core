package op

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
)

// SecretsManager is responsible for accessing the backend where secrets are stored
// and retrieving values for a given key.
type SecretsManager interface {
	Get(ctx context.Context, key string) (*secrets.Secret, errors.Error)
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
