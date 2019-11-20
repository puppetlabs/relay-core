package fn

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
)

var Library = NewMap(map[string]Descriptor{
	"jsonUnmarshal": DescriptorFuncs{
		DescriptionFunc: func() string { return "Unmarshals a JSON-encoded string into the specification" },
		PositionalInvokerFunc: func(args []interface{}) (Invoker, error) {
			if len(args) != 1 {
				return nil, &ArityError{Wanted: []int{1}, Got: len(args)}
			}

			fn := InvokerFunc(func(ctx context.Context) (m interface{}, err error) {
				var b []byte

				switch arg := args[0].(type) {
				case []byte:
					b = arg
				case string:
					b = []byte(arg)
				default:
					return nil, &PositionalArgError{
						Arg: 1,
						Cause: &UnexpectedTypeError{
							Wanted: []reflect.Type{
								reflect.TypeOf([]byte(nil)),
								reflect.TypeOf(""),
							},
							Got: reflect.TypeOf(arg),
						},
					}
				}

				err = json.Unmarshal(b, &m)
				return
			})
			return fn, nil
		},
	},
	"concat": DescriptorFuncs{
		DescriptionFunc: func() string { return "Concatenates string arguments into a single string" },
		PositionalInvokerFunc: func(args []interface{}) (Invoker, error) {
			if len(args) == 0 {
				return StaticInvoker(""), nil
			}

			fn := InvokerFunc(func(ctx context.Context) (m interface{}, err error) {
				strs := make([]string, len(args))
				for i, iarg := range args {
					switch arg := iarg.(type) {
					case []byte:
						strs[i] = string(arg)
					case string:
						strs[i] = arg
					default:
						return nil, &PositionalArgError{
							Arg: i + 1,
							Cause: &UnexpectedTypeError{
								Wanted: []reflect.Type{
									reflect.TypeOf([]byte(nil)),
									reflect.TypeOf(""),
								},
								Got: reflect.TypeOf(arg),
							},
						}
					}
				}

				return strings.Join(strs, ""), nil
			})
			return fn, nil
		},
	},
	"append": DescriptorFuncs{
		DescriptionFunc: func() string { return "Adds new items to a given array, returning a new array" },
		PositionalInvokerFunc: func(args []interface{}) (Invoker, error) {
			if len(args) < 2 {
				return nil, &ArityError{Wanted: []int{2}, Variadic: true, Got: len(args)}
			}

			fn := InvokerFunc(func(ctx context.Context) (m interface{}, err error) {
				base, ok := args[0].([]interface{})
				if !ok {
					return nil, &PositionalArgError{
						Arg: 1,
						Cause: &UnexpectedTypeError{
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
	},
})
