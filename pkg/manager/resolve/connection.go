package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type ConnectionTypeResolver struct {
	m model.ConnectionManager
}

var _ resolve.ConnectionTypeResolver = &ConnectionTypeResolver{}

func (ctr *ConnectionTypeResolver) ResolveConnection(ctx context.Context, typ, name string) (interface{}, error) {
	so, err := ctr.m.Get(ctx, typ, name)
	if err == model.ErrNotFound {
		return nil, &exprmodel.ConnectionNotFoundError{Type: typ, Name: name}
	} else if err != nil {
		return nil, err
	}

	return so.Attributes, nil
}

func NewConnectionTypeResolver(m model.ConnectionManager) *ConnectionTypeResolver {
	return &ConnectionTypeResolver{
		m: m,
	}
}
