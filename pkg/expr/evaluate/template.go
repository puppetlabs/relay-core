package evaluate

import (
	"context"

	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/pathlang"
	"github.com/puppetlabs/relay-core/pkg/expr/query"
)

type templater interface {
	eval.Indexable
	model.Expandable
}

type templateEnvironment map[string]templater

var _ templater = templateEnvironment(nil)

func (te templateEnvironment) Index(ctx context.Context, idx interface{}) (interface{}, error) {
	k, err := eval.StringValue(idx)
	if err != nil {
		return nil, err
	}

	r, ok := te[k]
	if !ok {
		return nil, &eval.UnknownKeyError{Key: k}
	}

	return r, nil
}

func (te templateEnvironment) Expand(ctx context.Context, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: te}, nil
	}

	rm := make(map[string]*model.Result, len(te))
	for key, v := range te {
		r, err := v.Expand(ctx, depth-1)
		if err != nil {
			return nil, err
		}

		rm[key] = r
	}

	return model.CombineResultMap(rm), nil
}

func evaluateTemplate(o *Options) func(ctx context.Context, s string, depth int, next model.Evaluator) (*model.Result, error) {
	env := make(templateEnvironment)

	// Data gets loaded first so names can't override.
	for name, resolver := range o.DataTypeResolvers {
		// We can't use the default resolver. It must be named.
		if name == "" {
			continue
		}

		env[name] = &pathlang.DataTypeResolverAdapter{DataTypeResolver: resolver}
	}

	env["secrets"] = &pathlang.SecretTypeResolverAdapter{SecretTypeResolver: o.SecretTypeResolver}
	env["connections"] = &pathlang.ConnectionTypeResolverAdapter{ConnectionTypeResolver: o.ConnectionTypeResolver}
	env["outputs"] = &pathlang.OutputTypeResolverAdapter{OutputTypeResolver: o.OutputTypeResolver}
	env["parameters"] = &pathlang.ParameterTypeResolverAdapter{ParameterTypeResolver: o.ParameterTypeResolver}
	env["status"] = &pathlang.StatusTypeResolverAdapter{StatusTypeResolver: o.StatusTypeResolver}
	env["statuses"] = &pathlang.StatusTypeResolverAdapter{StatusTypeResolver: o.StatusTypeResolver}

	return func(ctx context.Context, s string, depth int, next model.Evaluator) (*model.Result, error) {
		r, err := query.EvaluateQuery(ctx, model.DefaultEvaluator, query.PathTemplateLanguage(pathlang.WithFunctionMap{Map: o.FunctionMap}), env, s)
		if err != nil {
			return nil, err
		} else if !r.Complete() {
			return &model.Result{
				Value:        s,
				Unresolvable: r.Unresolvable,
			}, nil
		}

		return r, nil
	}
}
