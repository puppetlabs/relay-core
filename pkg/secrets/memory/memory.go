package memory

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
)

type SecretsManager struct {
	data map[string]string
}

func (m SecretsManager) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	if value, ok := m.data[key]; ok {
		return &secrets.Secret{
			Key:   key,
			Value: value,
		}, nil
	}

	return nil, errors.NewSecretsKeyNotFound(key)
}

func New(secrets map[string]string) *SecretsManager {
	return &SecretsManager{
		data: secrets,
	}
}
