package fnlib

import (
	"context"
	"reflect"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/query"
)

type pathArg struct {
	v   interface{}
	r   *model.Result
	err error
}

func path(ctx context.Context, ev model.Evaluator, objArg, qArg, defArg *pathArg) *model.Result {
	for _, arg := range []*pathArg{objArg, qArg, defArg} {
		if arg == nil {
			continue
		}

		arg.r, arg.err = ev.Evaluate(ctx, arg.v, 0)
		if arg.err != nil {
			return nil
		}
	}

	defUsable := defArg != nil

	qArg.r, qArg.err = model.EvaluateAll(ctx, ev, qArg.v)
	if qArg.err != nil || !qArg.r.Complete() {
		return nil
	}

	q, ok := qArg.r.Value.(string)
	if !ok {
		qArg.err = &fn.UnexpectedTypeError{
			Wanted: []reflect.Type{reflect.TypeOf("")},
			Got:    reflect.TypeOf(qArg.r.Value),
		}
		return nil
	}

	var vr *model.Result
	vr, objArg.err = query.EvaluateQuery(ctx, ev, query.PathLanguage(), objArg.v, q)
	if objArg.err != nil {
		if !errmark.Matches(objArg.err, notExistsRule) {
			defUsable = false
		}
	} else if vr.Complete() {
		return vr
	} else {
		// Note that we never set the object arg result to the result of the query
		// because it isn't representative of the entire data structure, which users
		// might find confusing.
		//
		// Instead, we just extend the specific part that couldn't be resolved.
		if objArg.r != nil {
			objArg.r.Extends(vr)
		}
		return nil
	}

	if !defUsable {
		return nil
	}

	// If we get this far, we should use the default. This means setting any
	// object error back to nil so that the whole function will be marked as not
	// resolvable if the default is not resolvable.
	objArg.err = nil

	defArg.r, defArg.err = model.EvaluateAll(ctx, ev, defArg.v)
	if defArg.err != nil || !defArg.r.Complete() {
		return nil
	}

	return defArg.r
}

var pathDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string {
		return "Looks up a value at a given path in an object, optionally returning a default value if the path does not exist"
	},
	PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, &fn.ArityError{Wanted: []int{2, 3}, Got: len(args)}
		}

		fn := fn.InvokerFunc(func(ctx context.Context) (*model.Result, error) {
			// For unresolved values, we want to show all of the arguments on
			// the way out.
			objArg := &pathArg{v: args[0]}
			qArg := &pathArg{v: args[1]}
			var defArg *pathArg
			if len(args) > 2 {
				defArg = &pathArg{v: args[2]}
			}

			r := path(ctx, ev, objArg, qArg, defArg)
			if r != nil {
				return r, nil
			}

			// Figure out what went wrong by traversing the arg intermediates.
			rs := make([]*model.Result, len(args))
			for i, arg := range []*pathArg{objArg, qArg, defArg} {
				if arg == nil {
					continue
				}

				if arg.err != nil {
					return nil, &fn.PositionalArgError{
						Arg:   i + 1,
						Cause: arg.err,
					}
				}

				rs[i] = arg.r
			}

			return model.CombineResultSlice(rs), nil
		})
		return fn, nil
	},
	KeywordInvokerFunc: func(ev model.Evaluator, args map[string]interface{}) (fn.Invoker, error) {
		for _, arg := range []string{"object", "query"} {
			if _, found := args[arg]; !found {
				return nil, &fn.KeywordArgError{Arg: arg, Cause: fn.ErrArgNotFound}
			}
		}

		fn := fn.InvokerFunc(func(ctx context.Context) (*model.Result, error) {
			objArg := &pathArg{v: args["object"]}
			qArg := &pathArg{v: args["query"]}
			var defArg *pathArg
			if arg, found := args["default"]; found {
				defArg = &pathArg{v: arg}
			}

			r := path(ctx, ev, objArg, qArg, defArg)
			if r != nil {
				return r, nil
			}

			// Figure out what went wrong by traversing the arg intermediates.
			rm := make(map[string]*model.Result)
			for key, arg := range map[string]*pathArg{
				"object":  objArg,
				"query":   qArg,
				"default": defArg,
			} {
				if arg == nil {
					continue
				}

				if arg.err != nil {
					return nil, &fn.KeywordArgError{
						Arg:   key,
						Cause: arg.err,
					}
				}

				rm[key] = arg.r
			}

			return model.CombineResultMap(rm), nil
		})
		return fn, nil
	},
}
