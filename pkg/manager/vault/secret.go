package vault

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type SecretManager struct {
	client *KVV2Client
}

var _ model.SecretManager = &SecretManager{}

func (m *SecretManager) List(ctx context.Context) ([]*model.Secret, error) {
	names, err := m.client.List(ctx)
	if err == model.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var l []*model.Secret

	for _, name := range names {
		s, err := m.Get(ctx, name)
		if err == model.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		l = append(l, s)
	}

	return l, nil
}

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
