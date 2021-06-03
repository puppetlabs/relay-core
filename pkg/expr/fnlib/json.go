package fnlib

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var (
	jsonMarshalDescriptor = fn.DescriptorFuncs{
		DescriptionFunc: func() string { return "Marshals arbitrary data into a JSON-encoded string" },
		PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
			if len(args) != 1 {
				return nil, &fn.ArityError{Wanted: []int{1}, Got: len(args)}
			}

			fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (interface{}, error) {
				b, err := json.Marshal(args[0])
				if err != nil {
					return nil, &fn.PositionalArgError{
						Arg:   1,
						Cause: err,
					}
				}

				return string(b), nil
			})
			return fn, nil
		},
	}
	jsonUnmarshalDescriptor = fn.DescriptorFuncs{
		DescriptionFunc: func() string { return "Unmarshals a JSON-encoded string into the specification" },
		PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
			if len(args) != 1 {
				return nil, &fn.ArityError{Wanted: []int{1}, Got: len(args)}
			}

			fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (m interface{}, err error) {
				var b []byte

				switch arg := args[0].(type) {
				case string:
					b = []byte(arg)
				default:
					return nil, &fn.PositionalArgError{
						Arg: 1,
						Cause: &fn.UnexpectedTypeError{
							Wanted: []reflect.Type{reflect.TypeOf("")},
							Got:    reflect.TypeOf(arg),
						},
					}
				}

				err = json.Unmarshal(b, &m)
				return
			})
			return fn, nil
		},
	}
)
