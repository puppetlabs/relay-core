package memory

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type SecretManager struct {
	secrets map[string]string
}

var _ model.SecretManager = &SecretManager{}

func (m *SecretManager) Get(ctx context.Context, name string) (*model.Secret, error) {
	value, found := m.secrets[name]
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.Secret{
		Name:  name,
		Value: value,
	}, nil
}

func NewSecretManager(secrets map[string]string) *SecretManager {
	m := make(map[string]string, len(secrets))
	for k, v := range secrets {
		m[k] = v
	}

	return &SecretManager{
		secrets: m,
	}
}
