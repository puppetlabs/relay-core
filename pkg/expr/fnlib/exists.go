package fnlib

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var existsDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Determines whether a value is set" },
	PositionalInvokerFunc: func(ev model.Evaluator, args []any) (fn.Invoker, error) {
		if len(args) != 1 {
			return nil, &fn.ArityError{Wanted: []int{1}, Got: len(args)}
		}

		fn := fn.InvokerFunc(func(ctx context.Context) (*model.Result, error) {
			r, err := model.EvaluateAll(ctx, ev, args[0])
			if errmark.Matches(err, notExistsRule) {
				return &model.Result{Value: false}, nil
			} else if err != nil {
				return nil, err
			} else if !r.Complete() {
				return model.CombineResultSlice([]*model.Result{r}), nil
			}

			return &model.Result{Value: true}, nil
		})
		return fn, nil
	},
}
