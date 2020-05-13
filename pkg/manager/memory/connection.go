package memory

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type ConnectionKey struct {
	Type, Name string
}

type ConnectionManager struct {
	connections map[ConnectionKey]map[string]interface{}
}

var _ model.ConnectionManager = &ConnectionManager{}

func (cm *ConnectionManager) Get(ctx context.Context, typ, name string) (*model.Connection, error) {
	attrs, found := cm.connections[ConnectionKey{Type: typ, Name: name}]
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.Connection{
		Type:       typ,
		Name:       name,
		Attributes: attrs,
	}, nil
}

func NewConnectionManager(m map[ConnectionKey]map[string]interface{}) *ConnectionManager {
	return &ConnectionManager{
		connections: m,
	}
}
