package resolve

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type OutputTypeResolver struct {
	m model.StepOutputGetterManager
}

var _ resolve.OutputTypeResolver = &OutputTypeResolver{}

func (otr *OutputTypeResolver) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	so, err := otr.m.Get(ctx, from, name)
	if err == model.ErrNotFound {
		return nil, &resolve.OutputNotFoundError{From: from, Name: name}
	} else if err != nil {
		return nil, err
	}

	return so.Value, nil
}

func NewOutputTypeResolver(m model.StepOutputGetterManager) *OutputTypeResolver {
	return &OutputTypeResolver{
		m: m,
	}
}
