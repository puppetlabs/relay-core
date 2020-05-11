package vault

import (
	"context"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type SecretManager struct {
	client *vaultapi.Client
	path   string
}

var _ model.SecretManager = &SecretManager{}

func (m *SecretManager) Get(ctx context.Context, name string) (*model.Secret, error) {
	vs, err := m.client.Logical().Read(secretPath(m.path, name))
	if err != nil {
		return nil, err
	} else if vs == nil {
		return nil, model.ErrNotFound
	}

	data, ok := vs.Data["data"].(map[string]interface{})
	if !ok {
		return nil, model.ErrNotFound
	}

	encoded, ok := data["value"].(string)
	if !ok {
		return nil, model.ErrNotFound
	}

	value, err := transfer.DecodeFromTransfer(encoded)
	if err != nil {
		return nil, err
	}

	return &model.Secret{
		Name:  name,
		Value: string(value),
	}, nil
}

func NewSecretManager(client *vaultapi.Client, path string) *SecretManager {
	return &SecretManager{
		client: client,
		path:   path,
	}
}

func secretPath(root, name string) string {
	return path.Join(root, name)
}
