package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type StatusTypeResolver struct {
	m model.ActionStatusGetterManager
}

var _ resolve.StatusTypeResolver = &StatusTypeResolver{}

func (str *StatusTypeResolver) ResolveStatus(ctx context.Context, name, property string) (bool, error) {
	step := &model.Step{
		Name: name,
	}

	as, err := str.m.Get(ctx, step)
	if err == model.ErrNotFound {
		return false, &exprmodel.StatusNotFoundError{Name: name, Property: property}
	} else if err != nil {
		return false, err
	}

	found, err := as.IsStatusProperty(model.StatusProperty(property))
	if err != nil {
		if err == model.ErrNotFound {
			return false, &exprmodel.StatusNotFoundError{Name: name, Property: property}
		} else {
			return false, err
		}
	}

	return found, nil
}

func NewStatusTypeResolver(m model.ActionStatusGetterManager) *StatusTypeResolver {
	return &StatusTypeResolver{
		m: m,
	}
}
