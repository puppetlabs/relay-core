package pathlang

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
)

type DataTypeResolverAdapter struct {
	resolve.DataTypeResolver
}

var (
	_ eval.Indexable   = &DataTypeResolverAdapter{}
	_ model.Expandable = &DataTypeResolverAdapter{}
)

func (dtra *DataTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	d, err := dtra.ResolveData(ctx)
	if errmark.Matches(err, errmark.RuleType(&model.DataNotFoundError{})) {
		return model.StaticExpandable(nil, model.Unresolvable{
			Data: []model.UnresolvableData{{}},
		}), nil
	} else if err != nil {
		return nil, err
	}

	return eval.Select(ctx, d, idx)
}

func (dtra *DataTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: dtra}, nil
	}

	d, err := dtra.ResolveData(ctx)
	if err != nil {
		return nil, err
	}

	return &model.Result{Value: d}, nil
}

type SecretTypeResolverAdapter struct {
	resolve.SecretTypeResolver
}

var (
	_ eval.Indexable   = &SecretTypeResolverAdapter{}
	_ model.Expandable = &SecretTypeResolverAdapter{}
)

func (stra *SecretTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	value, err := stra.ResolveSecret(ctx, name)
	if errmark.Matches(err, errmark.RuleType(&model.SecretNotFoundError{})) {
		return model.StaticExpandable("", model.Unresolvable{
			Secrets: []model.UnresolvableSecret{{Name: name}},
		}), nil
	} else if err != nil {
		return nil, err
	}

	return value, nil
}

func (stra *SecretTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: stra}, nil
	}

	m, err := stra.ResolveAllSecrets(ctx)
	if err != nil {
		return nil, err
	}

	cm := make(map[string]interface{}, len(m))
	for k, v := range m {
		cm[k] = v
	}

	return &model.Result{Value: cm}, nil
}

type typeOfConnectionTypeResolverAdapter struct {
	r   resolve.ConnectionTypeResolver
	typ string
}

var (
	_ eval.Indexable   = &typeOfConnectionTypeResolverAdapter{}
	_ model.Expandable = &typeOfConnectionTypeResolverAdapter{}
)

func (toctra *typeOfConnectionTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	value, err := toctra.r.ResolveConnection(ctx, toctra.typ, name)
	if errmark.Matches(err, errmark.RuleType(&model.ConnectionNotFoundError{})) {
		return model.StaticExpandable(nil, model.Unresolvable{
			Connections: []model.UnresolvableConnection{{Type: toctra.typ, Name: name}},
		}), nil
	} else if err != nil {
		return nil, err
	}

	return value, nil
}

func (toctra *typeOfConnectionTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: toctra}, nil
	}

	m, err := toctra.r.ResolveTypeOfConnections(ctx, toctra.typ)
	if err != nil {
		return nil, err
	}

	return &model.Result{Value: m}, nil
}

type ConnectionTypeResolverAdapter struct {
	resolve.ConnectionTypeResolver
}

var (
	_ eval.Indexable   = &ConnectionTypeResolverAdapter{}
	_ model.Expandable = &ConnectionTypeResolverAdapter{}
)

func (ctra *ConnectionTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	typ, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	return &typeOfConnectionTypeResolverAdapter{
		r:   ctra.ConnectionTypeResolver,
		typ: typ,
	}, nil
}

func (ctra *ConnectionTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: ctra}, nil
	}

	m, err := ctra.ResolveAllConnections(ctx)
	if err != nil {
		return nil, err
	}

	cm := make(map[string]interface{}, len(m))
	for k, v := range m {
		cm[k] = v
	}

	return &model.Result{Value: cm}, nil
}

type stepOutputTypeResolverAdapter struct {
	r    resolve.OutputTypeResolver
	from string
}

var (
	_ eval.Indexable   = &stepOutputTypeResolverAdapter{}
	_ model.Expandable = &stepOutputTypeResolverAdapter{}
)

func (sotra *stepOutputTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	value, err := sotra.r.ResolveOutput(ctx, sotra.from, name)
	if errmark.Matches(err, errmark.RuleType(&model.OutputNotFoundError{})) {
		return model.StaticExpandable(nil, model.Unresolvable{
			Outputs: []model.UnresolvableOutput{{From: sotra.from, Name: name}},
		}), nil
	} else if err != nil {
		return nil, err
	}

	return value, nil
}

func (sotra *stepOutputTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: sotra}, nil
	}

	m, err := sotra.r.ResolveStepOutputs(ctx, sotra.from)
	if err != nil {
		return nil, err
	}

	return &model.Result{Value: m}, nil
}

type OutputTypeResolverAdapter struct {
	resolve.OutputTypeResolver
}

var (
	_ eval.Indexable   = &OutputTypeResolverAdapter{}
	_ model.Expandable = &OutputTypeResolverAdapter{}
)

func (otra *OutputTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	from, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	return &stepOutputTypeResolverAdapter{
		r:    otra.OutputTypeResolver,
		from: from,
	}, nil
}

func (otra *OutputTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: otra}, nil
	}

	m, err := otra.ResolveAllOutputs(ctx)
	if err != nil {
		return nil, err
	}

	cm := make(map[string]interface{}, len(m))
	for k, v := range m {
		cm[k] = v
	}

	return &model.Result{Value: cm}, nil
}

type ParameterTypeResolverAdapter struct {
	resolve.ParameterTypeResolver
}

var (
	_ eval.Indexable   = &ParameterTypeResolverAdapter{}
	_ model.Expandable = &ParameterTypeResolverAdapter{}
)

func (ptra *ParameterTypeResolverAdapter) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	name, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	value, err := ptra.ResolveParameter(ctx, name)
	if errmark.Matches(err, errmark.RuleType(&model.ParameterNotFoundError{})) {
		return model.StaticExpandable(nil, model.Unresolvable{
			Parameters: []model.UnresolvableParameter{{Name: name}},
		}), nil
	} else if err != nil {
		return nil, err
	}

	return value, nil
}

func (ptra *ParameterTypeResolverAdapter) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: ptra}, nil
	}

	m, err := ptra.ResolveAllParameters(ctx)
	if err != nil {
		return nil, err
	}

	return &model.Result{Value: m}, nil
}
