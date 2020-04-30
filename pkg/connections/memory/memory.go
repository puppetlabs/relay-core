package memory

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConnectionsManager struct {
	data map[string]map[string]string
}

func (m ConnectionsManager) Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error) {
	return nil, nil
}

func New(conns map[string]map[string]string) *ConnectionsManager {
	return &ConnectionsManager{
		data: conns,
	}
}
