package specadapter

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/spec"
)

type ConnectionTypeResolver struct {
	m model.ConnectionManager
}

var _ spec.ConnectionTypeResolver = &ConnectionTypeResolver{}

func (ctr *ConnectionTypeResolver) ResolveAllConnections(ctx context.Context) (map[string]map[string]interface{}, error) {
	l, err := ctr.m.List(ctx)
	if err != nil {
		return nil, err
	} else if len(l) == 0 {
		return nil, nil
	}

	cm := make(map[string]map[string]interface{})

	for _, c := range l {
		tm, found := cm[c.Type]
		if !found {
			tm = make(map[string]interface{})
			cm[c.Type] = tm
		}

		tm[c.Name] = c.Attributes
	}

	return cm, nil
}

func (ctr *ConnectionTypeResolver) ResolveTypeOfConnections(ctx context.Context, typ string) (map[string]interface{}, error) {
	l, err := ctr.m.List(ctx)
	if err != nil {
		return nil, err
	}

	var tm map[string]interface{}

	for _, c := range l {
		if c.Type != typ {
			continue
		} else if tm == nil {
			tm = make(map[string]interface{})
		}

		tm[c.Name] = c.Attributes
	}

	return tm, nil
}

func (ctr *ConnectionTypeResolver) ResolveConnection(ctx context.Context, typ, name string) (interface{}, error) {
	c, err := ctr.m.Get(ctx, typ, name)
	if err == model.ErrNotFound {
		return nil, spec.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return c.Attributes, nil
}

func NewConnectionTypeResolver(m model.ConnectionManager) *ConnectionTypeResolver {
	return &ConnectionTypeResolver{
		m: m,
	}
}
