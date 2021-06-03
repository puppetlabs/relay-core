package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type OutputTypeResolver struct {
	m model.StepOutputGetterManager
}

var _ resolve.OutputTypeResolver = &OutputTypeResolver{}

func (otr *OutputTypeResolver) ResolveAllOutputs(ctx context.Context) (map[string]map[string]interface{}, error) {
	l, err := otr.m.List(ctx)
	if err != nil {
		return nil, err
	} else if len(l) == 0 {
		return nil, nil
	}

	om := make(map[string]map[string]interface{})

	for _, so := range l {
		sm, found := om[so.Step.Name]
		if !found {
			sm = make(map[string]interface{})
			om[so.Step.Name] = sm
		}

		sm[so.Name] = so.Value
	}

	return om, nil
}

func (otr *OutputTypeResolver) ResolveStepOutputs(ctx context.Context, from string) (map[string]interface{}, error) {
	l, err := otr.m.List(ctx)
	if err != nil {
		return nil, err
	}

	var sm map[string]interface{}

	for _, so := range l {
		if so.Step.Name != from {
			continue
		} else if sm == nil {
			sm = make(map[string]interface{})
		}

		sm[so.Name] = so.Value
	}

	return sm, nil
}

func (otr *OutputTypeResolver) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	so, err := otr.m.Get(ctx, from, name)
	if err == model.ErrNotFound {
		return nil, &exprmodel.OutputNotFoundError{From: from, Name: name}
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
