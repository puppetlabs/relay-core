package fn

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type Invoker interface {
	Invoke(ctx context.Context) (*model.Result, error)
}

type InvokerFunc func(ctx context.Context) (*model.Result, error)

var _ Invoker = InvokerFunc(nil)

func (fn InvokerFunc) Invoke(ctx context.Context) (*model.Result, error) {
	return fn(ctx)
}

func StaticInvoker(value interface{}) Invoker {
	return InvokerFunc(func(_ context.Context) (*model.Result, error) { return &model.Result{Value: value}, nil })
}

func EvaluatedPositionalInvoker(args []model.Evaluable, fn func(ctx context.Context, args []interface{}) (interface{}, error)) Invoker {
	return InvokerFunc(func(ctx context.Context) (*model.Result, error) {
		vs, err := model.EvaluateAllSlice(ctx, args)
		if err != nil {
			return nil, err
		}

		r := model.CombineResultSlice(vs)
		if !r.Complete() {
			return r, nil
		}

		rv, err := fn(ctx, r.Value.([]interface{}))
		if err != nil {
			return nil, err
		}

		return &model.Result{Value: rv}, nil
	})
}

func EvaluatedKeywordInvoker(args map[string]model.Evaluable, fn func(ctx context.Context, args map[string]interface{}) (interface{}, error)) Invoker {
	return InvokerFunc(func(ctx context.Context) (*model.Result, error) {
		vs, err := model.EvaluateAllMap(ctx, args)
		if err != nil {
			return nil, err
		}

		r := model.CombineResultMap(vs)
		if !r.Complete() {
			return r, nil
		}

		rv, err := fn(ctx, r.Value.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		return &model.Result{Value: rv}, nil
	})
}
