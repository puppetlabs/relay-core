package memory

import (
	"context"
	"path"

	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConnectionsManager struct {
	data map[string]map[string]string
}

func (m ConnectionsManager) Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error) {
	data, ok := m.data[path.Join(typ, name)]
	if !ok {
		return nil, errors.NewConnectionsTypeNameNotFound(typ, name)
	}

	return &connections.Connection{Spec: data}, nil
}

func New(conns map[string]map[string]string) *ConnectionsManager {
	return &ConnectionsManager{
		data: conns,
	}
}
