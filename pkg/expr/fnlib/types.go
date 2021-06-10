package fnlib

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var toStringDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Converts arbitrary scalar input data to a string" },
	PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
		if len(args) != 1 {
			return nil, &fn.ArityError{Wanted: []int{1}, Got: len(args)}
		}

		fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (interface{}, error) {
			arg, err := toString(args[0])
			if err != nil {
				return nil, &fn.PositionalArgError{
					Arg:   1,
					Cause: err,
				}
			}

			return arg, nil
		})
		return fn, nil
	},
}
