package specadapter

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/spec"
)

type StatusTypeResolver struct {
	m model.ActionStatusGetterManager
}

var _ spec.StatusTypeResolver = &StatusTypeResolver{}

func (str *StatusTypeResolver) ResolveStatus(ctx context.Context, action, property string) (bool, error) {
	step := &model.Step{
		Name: action,
	}

	as, err := str.m.Get(ctx, step)
	if err == model.ErrNotFound {
		return false, spec.ErrNotFound
	} else if err != nil {
		return false, err
	}

	found, err := as.IsStatusProperty(model.StatusProperty(property))
	if err != nil {
		if err == model.ErrNotFound {
			return false, spec.ErrNotFound
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
