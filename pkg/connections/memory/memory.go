package memory

import (
	"context"
	"path"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConnectionsManager struct {
	data map[string]map[string]transfer.JSONInterface
}

func (m ConnectionsManager) Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error) {
	data, ok := m.data[path.Join(typ, name)]
	if !ok {
		return nil, errors.NewConnectionsTypeNameNotFound(typ, name)
	}

	return &connections.Connection{Spec: data}, nil
}

func New(conns map[string]map[string]interface{}) *ConnectionsManager {
	newData := make(map[string]map[string]transfer.JSONInterface)
	for connPath, data := range conns {
		values := make(map[string]transfer.JSONInterface)

		for k, v := range data {
			values[k] = transfer.JSONInterface{Data: v}
		}

		newData[connPath] = values
	}

	return &ConnectionsManager{
		data: newData,
	}
}
