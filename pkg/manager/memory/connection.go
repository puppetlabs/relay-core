package memory

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ConnectionKey struct {
	Type, Name string
}

type ConnectionManager struct {
	connections map[ConnectionKey]map[string]interface{}
}

var _ model.ConnectionManager = &ConnectionManager{}

func (cm *ConnectionManager) List(ctx context.Context) ([]*model.Connection, error) {
	var l []*model.Connection

	for k, v := range cm.connections {
		l = append(l, &model.Connection{
			Type:       k.Type,
			Name:       k.Name,
			Attributes: v,
		})
	}

	return l, nil
}

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
