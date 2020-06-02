package fnlib

import (
	"context"
	"reflect"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/convert"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/fn"
)

var convertMarkdownDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Converts a string in markdown format to another applicable syntax" },
	PositionalInvokerFunc: func(args []interface{}) (fn.Invoker, error) {
		if len(args) != 2 {
			return nil, &fn.ArityError{Wanted: []int{2}, Variadic: false, Got: len(args)}
		}

		fn := fn.InvokerFunc(func(ctx context.Context) (m interface{}, err error) {
			to, found := args[0].(string)
			if !found {
				return nil, &fn.PositionalArgError{
					Arg: 0,
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf("")},
						Got:    reflect.TypeOf(args[0]),
					},
				}
			}

			switch md := args[1].(type) {
			case string:
				r, err := convert.ConvertMarkdown(convert.ConvertType(to), []byte(md))
				if err != nil {
					return nil, err
				}
				return string(r), nil
			default:
				return nil, &fn.PositionalArgError{
					Arg: 1,
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf("")},
						Got:    reflect.TypeOf(args[1]),
					},
				}
			}
		})
		return fn, nil
	},
	KeywordInvokerFunc: func(args map[string]interface{}) (fn.Invoker, error) {
		to, found := args["to"]
		if !found {
			return nil, &fn.KeywordArgError{Arg: "to", Cause: fn.ErrArgNotFound}
		}

		content, found := args["content"]
		if !found {
			return nil, &fn.KeywordArgError{Arg: "content", Cause: fn.ErrArgNotFound}
		}

		return fn.InvokerFunc(func(ctx context.Context) (interface{}, error) {
			ct, ok := to.(string)
			if !ok {
				return nil, &fn.KeywordArgError{
					Arg: "to",
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf("")},
						Got:    reflect.TypeOf(to),
					},
				}
			}

			switch md := content.(type) {
			case string:
				r, err := convert.ConvertMarkdown(convert.ConvertType(ct), []byte(md))
				if err != nil {
					return nil, err
				}
				return string(r), nil
			default:
				return nil, &fn.KeywordArgError{
					Arg: "content",
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf("")},
						Got:    reflect.TypeOf(content),
					},
				}
			}
		}), nil
	},
}
