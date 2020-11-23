package fnlib

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var coalesceDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string {
		return "Finds and returns the first resolvable non-null argument, returning null otherwise"
	},
	PositionalInvokerFunc: func(args []model.Evaluable) (fn.Invoker, error) {
		fn := fn.InvokerFunc(func(ctx context.Context) (*model.Result, error) {
			for _, arg := range args {
				r, err := arg.EvaluateAll(ctx)
				if err != nil {
					return nil, err
				} else if r.Complete() && r.Value != nil {
					return r, nil
				}
			}

			return &model.Result{Value: nil}, nil
		})
		return fn, nil
	},
}
