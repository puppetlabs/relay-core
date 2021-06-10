package fnlib

import (
	"context"
	"reflect"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var (
	equalsDescriptor = fn.DescriptorFuncs{
		DescriptionFunc: func() string { return "Checks if the left side equals the right side" },
		PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
			if len(args) != 2 {
				return nil, &fn.ArityError{Wanted: []int{2}, Got: len(args)}
			}

			fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (m interface{}, err error) {
				return reflect.DeepEqual(args[0], args[1]), nil
			})

			return fn, nil
		},
	}

	notEqualsDescriptor = fn.DescriptorFuncs{
		DescriptionFunc: func() string { return "Checks if the left side does not equal the right side" },
		PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
			if len(args) != 2 {
				return nil, &fn.ArityError{Wanted: []int{2}, Got: len(args)}
			}

			fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (m interface{}, err error) {
				return !reflect.DeepEqual(args[0], args[1]), nil
			})

			return fn, nil
		},
	}
)
