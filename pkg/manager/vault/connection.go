package vault

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type ConnectionManager struct {
	client *KVV2Client
}

var _ model.ConnectionManager = &ConnectionManager{}

func (m *ConnectionManager) Get(ctx context.Context, typ, name string) (*model.Connection, error) {
	connectionID, err := m.client.In(typ, name).ReadString(ctx)
	if err != nil {
		return nil, err
	}

	keys, err := m.client.In(connectionID).List(ctx)
	if err != nil {
		return nil, err
	}

	attrs := make(map[string]string, len(keys))
	for _, key := range keys {
		value, err := m.client.In(connectionID, key).ReadString(ctx)
		if err == model.ErrNotFound {
			// Deleted from under us?
			continue
		} else if err != nil {
			return nil, err
		}

		attrs[key] = value
	}

	return &model.Connection{
		Type:       typ,
		Name:       name,
		Attributes: attrs,
	}, nil
}

func NewConnectionManager(client *KVV2Client) *ConnectionManager {
	return &ConnectionManager{
		client: client,
	}
}
