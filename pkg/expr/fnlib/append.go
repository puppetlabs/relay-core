package fnlib

import (
	"context"
	"reflect"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var appendDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Adds new items to a given array, returning a new array" },
	PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
		if len(args) < 2 {
			return nil, &fn.ArityError{Wanted: []int{2}, Variadic: true, Got: len(args)}
		}

		fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (interface{}, error) {
			base, ok := args[0].([]interface{})
			if !ok {
				return nil, &fn.PositionalArgError{
					Arg: 1,
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{
							reflect.TypeOf([]interface{}(nil)),
						},
						Got: reflect.TypeOf(args[0]),
					},
				}
			}

			new := append([]interface{}{}, base...)
			new = append(new, args[1:]...)
			return new, nil
		})
		return fn, nil
	},
}
