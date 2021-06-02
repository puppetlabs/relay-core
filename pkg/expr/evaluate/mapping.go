package evaluate

import (
	"context"
	"reflect"
	"strings"

	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/query"
)

func evaluateType(ctx context.Context, tm map[string]interface{}, o *Options) (*model.Result, error) {
	switch tm["$type"] {
	case "Data":
		// No name indicates that we should the default resolver, which is
		// always the empty string.
		name, _ := tm["name"].(string)

		q, ok := tm["query"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Data", Cause: &FieldNotFoundError{Name: "query"}}
		}

		resolver, found := o.DataTypeResolvers[name]
		if !found {
			return nil, &InvalidTypeError{Type: "Data", Cause: &DataResolverNotFoundError{Name: name}}
		}

		d, err := resolver.ResolveData(ctx)
		if err != nil {
			return nil, &InvalidTypeError{Type: "Data", Cause: err}
		}

		r, err := query.EvaluateQuery(ctx, model.DefaultEvaluator, query.PathLanguage(), d, q)
		if err != nil {
			return nil, &InvalidTypeError{Type: "Data", Cause: &DataQueryError{Query: q, Cause: err}}
		} else if !r.Complete() {
			return &model.Result{
				Value:        tm,
				Unresolvable: r.Unresolvable,
			}, nil
		}

		return &model.Result{Value: r.Value}, nil
	case "Secret":
		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Secret", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := o.SecretTypeResolver.ResolveSecret(ctx, name)
		if serr, ok := err.(*model.SecretNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Secrets: []model.UnresolvableSecret{
					{Name: serr.Name},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Secret", Cause: err}
		}

		return &model.Result{Value: value}, nil
	case "Connection":
		connectionType, ok := tm["type"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Connection", Cause: &FieldNotFoundError{Name: "type"}}
		}

		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Connection", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := o.ConnectionTypeResolver.ResolveConnection(ctx, connectionType, name)
		if oerr, ok := err.(*model.ConnectionNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Connections: []model.UnresolvableConnection{
					{Type: oerr.Type, Name: oerr.Name},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Connection", Cause: err}
		}

		return &model.Result{Value: value}, nil
	case "Output":
		from, ok := tm["from"].(string)
		if !ok {
			// Fall back to old syntax.
			//
			// TODO: Remove this in a second version.
			from, ok = tm["taskName"].(string)
			if !ok {
				return nil, &InvalidTypeError{Type: "Output", Cause: &FieldNotFoundError{Name: "from"}}
			}
		}

		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Output", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := o.OutputTypeResolver.ResolveOutput(ctx, from, name)
		if oerr, ok := err.(*model.OutputNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Outputs: []model.UnresolvableOutput{
					{From: oerr.From, Name: oerr.Name},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Output", Cause: err}
		}

		return &model.Result{Value: value}, nil
	case "Parameter":
		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Parameter", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := o.ParameterTypeResolver.ResolveParameter(ctx, name)
		if perr, ok := err.(*model.ParameterNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Parameters: []model.UnresolvableParameter{
					{Name: perr.Name},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Parameter", Cause: err}
		}

		return &model.Result{Value: value}, nil
	case "Answer":
		askRef, ok := tm["askRef"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Answer", Cause: &FieldNotFoundError{Name: "askRef"}}
		}

		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Answer", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := o.AnswerTypeResolver.ResolveAnswer(ctx, askRef, name)
		if oerr, ok := err.(*model.AnswerNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Answers: []model.UnresolvableAnswer{
					{AskRef: oerr.AskRef, Name: oerr.Name},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Answer", Cause: err}
		}

		return &model.Result{Value: value}, nil
	default:
		return &model.Result{Value: tm}, nil
	}
}

func evaluateEncoding(ctx context.Context, em map[string]interface{}, next model.Evaluator) (*model.Result, error) {
	ty, ok := em["$encoding"].(string)
	if !ok {
		return &model.Result{Value: em}, nil
	}

	dr, err := model.EvaluateAll(ctx, next, em["data"])
	if err != nil {
		return nil, &InvalidEncodingError{Type: ty, Cause: err}
	} else if !dr.Complete() {
		r := &model.Result{
			Value: map[string]interface{}{
				"$encoding": ty,
				"data":      dr.Value,
			},
		}
		r.Extends(dr)
		return r, nil
	}

	data, ok := dr.Value.(string)
	if !ok {
		return nil, &InvalidEncodingError{
			Type: ty,
			Cause: &fn.UnexpectedTypeError{
				Wanted: []reflect.Type{reflect.TypeOf("")},
				Got:    reflect.TypeOf(dr.Value),
			},
		}
	}

	decoded, err := transfer.JSON{
		EncodingType: transfer.EncodingType(ty),
		Data:         data,
	}.Decode()
	if err != nil {
		return nil, &InvalidEncodingError{Type: ty, Cause: err}
	}

	return &model.Result{Value: string(decoded)}, nil
}

func evaluateInvocation(ctx context.Context, im map[string]interface{}, next model.Evaluator, funcs fn.Map) (*model.Result, error) {
	var key string
	var value interface{}
	for key, value = range im {
	}

	name := strings.TrimPrefix(key, "$fn.")

	descriptor, err := funcs.Descriptor(name)
	if err != nil {
		return &model.Result{
			Value: im,
			Unresolvable: model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			},
		}, nil
	}

	var invoker fn.Invoker

	// Evaluate one level to determine whether we should do a positional or
	// keyword invocation.
	a, err := next.Evaluate(ctx, value, 1)
	if err != nil {
		return nil, err
	} else if !a.Complete() {
		// The top level couldn't be resolved, so we'll pass it in unmodified as
		// a single-argument parameter.
		invoker, err = descriptor.PositionalInvoker(next, []interface{}{value})
	} else {
		switch ra := a.Value.(type) {
		case []interface{}:
			invoker, err = descriptor.PositionalInvoker(next, ra)
		case map[string]interface{}:
			invoker, err = descriptor.KeywordInvoker(next, ra)
		default:
			invoker, err = descriptor.PositionalInvoker(next, []interface{}{ra})
		}
	}
	if err != nil {
		return &model.Result{
			Value: im,
			Unresolvable: model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			},
		}, nil
	}

	a, err = invoker.Invoke(ctx)
	if err != nil {
		return nil, &model.InvocationError{Name: name, Cause: err}
	} else if !a.Complete() {
		r := &model.Result{
			Value: map[string]interface{}{
				key: a.Value,
			},
		}
		r.Extends(a)
		return r, nil
	}

	return a, nil
}

func evaluateMap(o *Options) func(ctx context.Context, m map[string]interface{}, depth int, next model.Evaluator) (*model.Result, error) {
	return func(ctx context.Context, m map[string]interface{}, depth int, next model.Evaluator) (*model.Result, error) {
		if _, ok := m["$type"]; ok {
			return evaluateType(ctx, m, o)
		} else if _, ok := m["$encoding"]; ok {
			return evaluateEncoding(ctx, m, next)
		} else if len(m) == 1 {
			var first string
			for first = range m {
			}

			if strings.HasPrefix(first, "$fn.") {
				return evaluateInvocation(ctx, m, next, o.FunctionMap)
			}
		}

		return model.DefaultVisitor.VisitMap(ctx, m, depth, next)
	}
}
