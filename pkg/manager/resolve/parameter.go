package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type ParameterTypeResolver struct {
	m model.ParameterGetterManager
}

var _ resolve.ParameterTypeResolver = &ParameterTypeResolver{}

func (ptr *ParameterTypeResolver) ResolveAllParameters(ctx context.Context) (map[string]interface{}, error) {
	l, err := ptr.m.List(ctx)
	if err != nil {
		return nil, err
	} else if len(l) == 0 {
		return nil, nil
	}

	pm := make(map[string]interface{}, len(l))

	for _, p := range l {
		pm[p.Name] = p.Value
	}

	return pm, nil
}

func (ptr *ParameterTypeResolver) ResolveParameter(ctx context.Context, name string) (interface{}, error) {
	p, err := ptr.m.Get(ctx, name)
	if err == model.ErrNotFound {
		return nil, &exprmodel.ParameterNotFoundError{Name: name}
	} else if err != nil {
		return nil, err
	}

	return p.Value, nil
}

func NewParameterTypeResolver(m model.ParameterGetterManager) *ParameterTypeResolver {
	return &ParameterTypeResolver{
		m: m,
	}
}
