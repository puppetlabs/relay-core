package vault

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type SecretManager struct {
	client *KVV2Client
}

var _ model.SecretManager = &SecretManager{}

func (m *SecretManager) Get(ctx context.Context, name string) (*model.Secret, error) {
	value, err := m.client.In(name).ReadString(ctx)
	if err != nil {
		return nil, err
	}

	return &model.Secret{
		Name:  name,
		Value: value,
	}, nil
}

func NewSecretManager(client *KVV2Client) *SecretManager {
	return &SecretManager{
		client: client,
	}
}
