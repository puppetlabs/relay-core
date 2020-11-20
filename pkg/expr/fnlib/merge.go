package fnlib

import (
	"context"
	"fmt"
	"reflect"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

func merge(dst, src map[string]interface{}, deep bool) {
	for k, v := range src {
		if deep {
			if dm, ok := dst[k].(map[string]interface{}); ok {
				if sm, ok := v.(map[string]interface{}); ok {
					merge(dm, sm, deep)
					continue
				}
			}
		}

		dst[k] = v
	}
}

func mergeCast(os []interface{}, errFn func(i int, err error) error) ([]map[string]interface{}, error) {
	objs := make([]map[string]interface{}, len(os))
	for i, o := range os {
		obj, ok := o.(map[string]interface{})
		if !ok {
			return nil, errFn(i, &fn.UnexpectedTypeError{
				Wanted: []reflect.Type{reflect.TypeOf(map[string]interface{}(nil))},
				Got:    reflect.TypeOf(o),
			})
		}

		objs[i] = obj
	}
	return objs, nil
}

var mergeDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string {
		return `Merges a series of objects, with each object overwriting prior entries.

Merges are performed deeply by default. Use the keyword form and set mode: shallow to perform a shallow merge.`
	},
	PositionalInvokerFunc: func(args []model.Evaluable) (fn.Invoker, error) {
		if len(args) == 0 {
			return fn.StaticInvoker(map[string]interface{}{}), nil
		}

		return fn.EvaluatedPositionalInvoker(args, func(ctx context.Context, args []interface{}) (interface{}, error) {
			objs, err := mergeCast(args, func(i int, err error) error {
				return &fn.PositionalArgError{
					Arg:   i + 1,
					Cause: err,
				}
			})
			if err != nil {
				return nil, err
			}

			r := make(map[string]interface{})
			for _, obj := range objs {
				merge(r, obj, true)
			}
			return r, nil
		}), nil
	},
	KeywordInvokerFunc: func(args map[string]model.Evaluable) (fn.Invoker, error) {
		if _, found := args["objects"]; !found {
			return nil, &fn.KeywordArgError{Arg: "objects", Cause: fn.ErrArgNotFound}
		}

		return fn.EvaluatedKeywordInvoker(args, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			mode, found := args["mode"]
			if !found {
				mode = "deep"
			}

			var deep bool
			switch mode {
			case "deep":
				deep = true
			case "shallow":
				deep = false
			default:
				return nil, &fn.KeywordArgError{
					Arg:   "mode",
					Cause: fmt.Errorf(`unexpected value %q, wanted one of "deep" or "shallow"`, mode),
				}
			}

			os, ok := args["objects"].([]interface{})
			if !ok {
				return nil, &fn.KeywordArgError{
					Arg: "objects",
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf([]interface{}(nil))},
						Got:    reflect.TypeOf(args["objects"]),
					},
				}
			}

			objs, err := mergeCast(os, func(i int, err error) error {
				return fmt.Errorf("array index %d: %+v", i, err)
			})
			if err != nil {
				return nil, &fn.KeywordArgError{
					Arg:   "objects",
					Cause: err,
				}
			}

			r := make(map[string]interface{})
			for _, obj := range objs {
				merge(r, obj, deep)
			}
			return r, nil
		}), nil
	},
}
