package evaluate

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/mitchellh/mapstructure"
	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
	jsonpathtemplate "github.com/puppetlabs/leg/jsonutil/pkg/jsonpath/template"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
)

type Language int

const (
	LanguagePath Language = 1 + iota
	LanguageJSONPath
	LanguageJSONPathTemplate
)

type InvokeFunc func(ctx context.Context, i fn.Invoker) (*model.Result, error)

type Evaluator struct {
	lang                   Language
	invoke                 InvokeFunc
	resultMapper           model.ResultMapper
	dataTypeResolver       resolve.DataTypeResolver
	secretTypeResolver     resolve.SecretTypeResolver
	connectionTypeResolver resolve.ConnectionTypeResolver
	outputTypeResolver     resolve.OutputTypeResolver
	parameterTypeResolver  resolve.ParameterTypeResolver
	answerTypeResolver     resolve.AnswerTypeResolver
	invocationResolver     resolve.InvocationResolver
}

func (e *Evaluator) ScopeTo(tree parse.Tree) *ScopedEvaluator {
	return &ScopedEvaluator{parent: e, tree: tree}
}

func (e *Evaluator) Copy(opts ...Option) *Evaluator {
	if len(opts) == 0 {
		return e
	}

	ne := &Evaluator{}
	*ne = *e

	for _, opt := range opts {
		opt(ne)
	}

	return ne
}

func (e *Evaluator) Evaluate(ctx context.Context, tree parse.Tree, depth int) (*model.Result, error) {
	r, err := e.evaluate(ctx, tree, depth)
	if err != nil {
		return nil, err
	}

	return e.resultMapper.MapResult(ctx, r)
}

func (e *Evaluator) EvaluateAll(ctx context.Context, tree parse.Tree) (*model.Result, error) {
	return e.Evaluate(ctx, tree, -1)
}

func (e *Evaluator) EvaluateInto(ctx context.Context, tree parse.Tree, target interface{}) (model.Unresolvable, error) {
	var u model.Unresolvable

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructureHookFunc(ctx, e, &u),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339Nano),
		),
		ZeroFields: true,
		Result:     target,
		TagName:    "spec",
	})
	if err != nil {
		return u, err
	}

	return u, d.Decode(tree)
}

func (e *Evaluator) EvaluateQuery(ctx context.Context, tree parse.Tree, query string) (*model.Result, error) {
	r := &model.Result{}

	var pl gval.Language
	switch e.lang {
	case LanguagePath:
		pl = gval.NewLanguage(
			gval.Base(),
			gval.VariableSelector(variableSelector(e, r)),
		)
	case LanguageJSONPath:
		pl = gval.NewLanguage(
			jsonpathtemplate.ExpressionLanguage(),
			gval.VariableSelector(jsonpath.VariableSelector(variableVisitor(e, r))),
		)
	case LanguageJSONPathTemplate:
		pl = jsonpathtemplate.TemplateLanguage(
			jsonpathtemplate.WithExpressionLanguageVariableVisitor(variableVisitor(e, r)),
			jsonpathtemplate.WithFormatter(func(v interface{}) (string, error) {
				rv, err := e.evaluate(ctx, v, -1)
				if err != nil {
					return "", err
				} else if !rv.Complete() {
					r.Extends(rv)
				} else {
					v = rv.Value
				}

				return jsonpathtemplate.DefaultFormatter(v)
			}),
		)
	default:
		return nil, ErrUnsupportedLanguage
	}

	path, err := pl.NewEvaluable(query)
	if err != nil {
		return nil, err
	}

	v, err := path(ctx, tree)
	if err != nil {
		return nil, err
	}

	er, err := e.evaluate(ctx, v, -1)
	if err != nil {
		return nil, err
	}

	// Add any other unresolved paths in here (provided by the variable selector).
	er.Extends(r)

	return e.resultMapper.MapResult(ctx, er)
}

func (e *Evaluator) evaluateType(ctx context.Context, tm map[string]interface{}) (*model.Result, error) {
	switch tm["$type"] {
	case "Data":
		query, ok := tm["query"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Data", Cause: &FieldNotFoundError{Name: "query"}}
		}

		value, err := e.dataTypeResolver.ResolveData(ctx, query)
		if serr, ok := err.(*model.DataNotFoundError); ok {
			return &model.Result{
				Value: tm,
				Unresolvable: model.Unresolvable{Data: []model.UnresolvableData{
					{Query: serr.Query},
				}},
			}, nil
		} else if err != nil {
			return nil, &InvalidTypeError{Type: "Data", Cause: err}
		}

		return &model.Result{Value: value}, nil
	case "Secret":
		name, ok := tm["name"].(string)
		if !ok {
			return nil, &InvalidTypeError{Type: "Secret", Cause: &FieldNotFoundError{Name: "name"}}
		}

		value, err := e.secretTypeResolver.ResolveSecret(ctx, name)
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

		value, err := e.connectionTypeResolver.ResolveConnection(ctx, connectionType, name)
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

		value, err := e.outputTypeResolver.ResolveOutput(ctx, from, name)
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

		value, err := e.parameterTypeResolver.ResolveParameter(ctx, name)
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

		value, err := e.answerTypeResolver.ResolveAnswer(ctx, askRef, name)
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

func (e *Evaluator) evaluateEncoding(ctx context.Context, em map[string]interface{}) (*model.Result, error) {
	ty, ok := em["$encoding"].(string)
	if !ok {
		return &model.Result{Value: em}, nil
	}

	dr, err := e.evaluate(ctx, em["data"], -1)
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

func (e *Evaluator) evaluateInvocation(ctx context.Context, im map[string]interface{}) (*model.Result, error) {
	var key string
	var value interface{}
	for key, value = range im {
	}

	name := strings.TrimPrefix(key, "$fn.")

	var invoker fn.Invoker

	// Evaluate one level to determine whether we should do a positional or
	// keyword invocation.
	a, err := e.evaluate(ctx, value, 1)
	if err != nil {
		return nil, err
	} else if !a.Complete() {
		// The top level couldn't be resolved, so we'll pass it in unmodified as
		// a single-argument parameter.
		invoker, err = e.invocationResolver.ResolveInvocationPositional(ctx, name, []model.Evaluable{e.ScopeTo(value).Copy(WithLanguage(LanguagePath))})
	} else {
		switch ra := a.Value.(type) {
		case []interface{}:
			args := make([]model.Evaluable, len(ra))
			for i, value := range ra {
				args[i] = e.ScopeTo(value).Copy(WithLanguage(LanguagePath))
			}

			invoker, err = e.invocationResolver.ResolveInvocationPositional(ctx, name, args)
		case map[string]interface{}:
			args := make(map[string]model.Evaluable, len(ra))
			for key, value := range ra {
				args[key] = e.ScopeTo(value).Copy(WithLanguage(LanguagePath))
			}

			invoker, err = e.invocationResolver.ResolveInvocation(ctx, name, args)
		default:
			invoker, err = e.invocationResolver.ResolveInvocationPositional(ctx, name, []model.Evaluable{e.ScopeTo(ra).Copy(WithLanguage(LanguagePath))})
		}
	}
	if ierr, ok := err.(*model.FunctionResolutionError); ok {
		return &model.Result{
			Value: im,
			Unresolvable: model.Unresolvable{Invocations: []model.UnresolvableInvocation{
				{Name: ierr.Name, Cause: ierr.Cause},
			}},
		}, nil
	} else if err != nil {
		return nil, &InvalidInvocationError{Name: name, Cause: err}
	}

	a, err = e.invoke(ctx, invoker)
	if err != nil {
		return nil, &InvocationError{Name: name, Cause: err}
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

func (e *Evaluator) evaluateUnchecked(ctx context.Context, v interface{}, depth int) (*model.Result, error) {
	if depth == 0 {
		return &model.Result{Value: v}, nil
	}

	switch vt := v.(type) {
	case []interface{}:
		if depth == 1 {
			return &model.Result{Value: v}, nil
		}

		r := &model.Result{}
		l := make([]interface{}, len(vt))
		for i, v := range vt {
			nv, err := e.evaluate(ctx, v, depth-1)
			if err != nil {
				return nil, &PathEvaluationError{
					Path:  strconv.Itoa(i),
					Cause: err,
				}
			}

			r.Extends(nv)
			l[i] = nv.Value
		}

		r.Value = l
		return r, nil
	case map[string]interface{}:
		if _, ok := vt["$type"]; ok {
			return e.evaluateType(ctx, vt)
		} else if _, ok := vt["$encoding"]; ok {
			return e.evaluateEncoding(ctx, vt)
		} else if len(vt) == 1 {
			var first string
			for first = range vt {
			}

			if strings.HasPrefix(first, "$fn.") {
				return e.evaluateInvocation(ctx, vt)
			}
		}

		if depth == 1 {
			return &model.Result{Value: v}, nil
		}

		r := &model.Result{}
		m := make(map[string]interface{}, len(vt))
		for k, v := range vt {
			nv, err := e.evaluate(ctx, v, depth-1)
			if err != nil {
				return nil, &PathEvaluationError{Path: k, Cause: err}
			}

			r.Extends(nv)
			m[k] = nv.Value
		}

		r.Value = m
		return r, nil
	default:
		return &model.Result{Value: v}, nil
	}
}

func (e *Evaluator) evaluate(ctx context.Context, v interface{}, depth int) (*model.Result, error) {
	candidate, err := e.evaluateUnchecked(ctx, v, depth)
	if err != nil {
		return nil, err
	}

	switch candidate.Value.(type) {
	// Valid JSON types per https://golang.org/pkg/encoding/json/:
	case bool, float64, string, []interface{}, map[string]interface{}, nil:
		return candidate, nil
	// We support a set of additional YAML scalar(-ish) types decoded by
	// gopkg.in/yaml.v3.
	case int, int64, uint, uint64, time.Time:
		return candidate, nil
	default:
		return nil, &UnsupportedValueError{Type: reflect.TypeOf(candidate.Value)}
	}
}

func NewEvaluator(opts ...Option) *Evaluator {
	e := &Evaluator{
		lang:                   LanguagePath,
		invoke:                 func(ctx context.Context, i fn.Invoker) (*model.Result, error) { return i.Invoke(ctx) },
		resultMapper:           model.IdentityResultMapper,
		dataTypeResolver:       resolve.NoOpDataTypeResolver,
		secretTypeResolver:     resolve.NoOpSecretTypeResolver,
		connectionTypeResolver: resolve.NoOpConnectionTypeResolver,
		outputTypeResolver:     resolve.NoOpOutputTypeResolver,
		parameterTypeResolver:  resolve.NoOpParameterTypeResolver,
		answerTypeResolver:     resolve.NoOpAnswerTypeResolver,
		invocationResolver:     resolve.NewDefaultMemoryInvocationResolver(),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}
