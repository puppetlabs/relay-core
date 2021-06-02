package evaluate_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/leg/gvalutil/pkg/template"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/expr/query"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/stretchr/testify/require"
)

type randomOrder []interface{}

type test struct {
	Name                 string
	Data                 string
	Opts                 []evaluate.Option
	Depth                int
	QueryLanguage        query.Language
	Query                string
	Into                 interface{}
	ExpectedValue        interface{}
	ExpectedUnresolvable model.Unresolvable
	ExpectedError        error
}

func (tt test) Run(t *testing.T) {
	ctx := context.Background()

	tree, err := parse.ParseJSONString(tt.Data)
	require.NoError(t, err)

	ev := evaluate.NewEvaluator(tt.Opts...)

	check := func(t *testing.T, err error) {
		if tt.ExpectedError != nil {
			require.Equal(t, tt.ExpectedError, err)
		} else {
			require.NoError(t, err)
		}
	}

	var v interface{}
	var u model.Unresolvable
	if tt.Query != "" {
		lang := tt.QueryLanguage
		if lang == nil {
			lang = query.PathLanguage()
		}

		r, err := query.EvaluateQuery(ctx, ev, lang, tree, tt.Query)
		check(t, err)

		if r != nil {
			v = r.Value
			u = r.Unresolvable
		}
	} else if tt.Into != nil {
		u, err = model.EvaluateInto(ctx, ev, tree, tt.Into)
		check(t, err)

		v = tt.Into
	} else {
		depth := tt.Depth
		if depth == 0 {
			depth = -1
		}

		r, err := ev.Evaluate(ctx, tree, depth)
		check(t, err)

		if r != nil {
			v = r.Value
			u = r.Unresolvable
		}
	}

	expected := tt.ExpectedValue
	if ro, ok := expected.(randomOrder); ok {
		expected = []interface{}(ro)

		// Requests sorting before continuing.
		if actual, ok := v.([]interface{}); ok {
			sort.Slice(actual, func(i, j int) bool {
				return fmt.Sprintf("%T %v", actual[i], actual[i]) < fmt.Sprintf("%T %v", actual[j], actual[j])
			})
		}
	}

	require.Equal(t, expected, v)
	require.Equal(t, tt.ExpectedUnresolvable, u)
}

type tests []test

func (tts tests) RunAll(t *testing.T) {
	for _, tt := range tts {
		t.Run(tt.Name, tt.Run)
	}
}

func TestEvaluate(t *testing.T) {
	fns := fn.NewMap(map[string]fn.Descriptor{
		"foo": fn.DescriptorFuncs{
			PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
				return fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (interface{}, error) {
					return fmt.Sprintf("~~%v~~", args), nil
				}), nil
			},
			KeywordInvokerFunc: func(ev model.Evaluator, args map[string]interface{}) (fn.Invoker, error) {
				return fn.EvaluatedKeywordInvoker(ev, args, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
					return fmt.Sprintf("~~%v~~", args["whiz"]), nil
				}), nil
			},
		},
	})

	tests{
		{
			Name:          "literal",
			Data:          `{"foo": "bar"}`,
			ExpectedValue: map[string]interface{}{"foo": "bar"},
		},
		{
			Name: "invalid data resolver",
			Data: `{"baz": {"$type": "Data", "query": "foo.bar"}}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "baz",
				Cause: &evaluate.InvalidTypeError{
					Type:  "Data",
					Cause: &evaluate.DataResolverNotFoundError{},
				},
			},
		},
		{
			Name: "invalid data resolver in template",
			Data: `{"baz": "${event.foo.bar}"}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "baz",
				Cause: &template.EvaluationError{
					Start: "${",
					Cause: &model.PathEvaluationError{
						Path: "event",
						Cause: &eval.UnknownKeyError{
							Key: "event",
						},
					},
				},
			},
		},
		{
			Name: "unresolvable secret",
			Data: `{"foo": {"$type": "Secret", "name": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONSecret("bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable secret in template",
			Data: `{"foo": "${secrets.bar}"}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${secrets.bar}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable connection",
			Data: `{"foo": {"$type": "Connection", "type": "blort", "name": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONConnection("blort", "bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Connections: []model.UnresolvableConnection{
					{Type: "blort", Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable connection in template",
			Data: `{"foo": "${connections.blort.bar}"}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${connections.blort.bar}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Connections: []model.UnresolvableConnection{
					{Type: "blort", Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable output",
			Data: `{"foo": {"$type": "Output", "from": "baz", "name": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONOutput("baz", "bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Outputs: []model.UnresolvableOutput{
					{From: "baz", Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable output in template",
			Data: `{"foo": "${outputs.baz.bar}"}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${outputs.baz.bar}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Outputs: []model.UnresolvableOutput{
					{From: "baz", Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable parameter",
			Data: `{"foo": {"$type": "Parameter", "name": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONParameter("bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable parameter in template",
			Data: `{"foo": "${parameters.bar}"}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${parameters.bar}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "invalid data",
			Data: `{"foo": [{"$type": "Data"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidTypeError{
						Type:  "Data",
						Cause: &evaluate.FieldNotFoundError{Name: "query"},
					},
				},
			},
		},
		{
			Name: "invalid secret",
			Data: `{"foo": [{"$type": "Secret"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidTypeError{
						Type:  "Secret",
						Cause: &evaluate.FieldNotFoundError{Name: "name"},
					},
				},
			},
		},
		{
			Name: "invalid connection",
			Data: `{"foo": [{"$type": "Connection", "name": "foo"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidTypeError{
						Type:  "Connection",
						Cause: &evaluate.FieldNotFoundError{Name: "type"},
					},
				},
			},
		},
		{
			Name: "invalid output",
			Data: `{"foo": [{"$type": "Output", "name": "foo"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidTypeError{
						Type:  "Output",
						Cause: &evaluate.FieldNotFoundError{Name: "from"},
					},
				},
			},
		},
		{
			Name: "invalid parameter",
			Data: `{"foo": [{"$type": "Parameter"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidTypeError{
						Type:  "Parameter",
						Cause: &evaluate.FieldNotFoundError{Name: "name"},
					},
				},
			},
		},
		{
			Name: "invalid encoding",
			Data: `{"foo": [{"$encoding": "base32", "data": "nooo"}]}`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &evaluate.InvalidEncodingError{
						Type:  "base32",
						Cause: transfer.ErrUnknownEncodingType,
					},
				},
			},
		},
		{
			Name: "data query error",
			Data: `{
				"data": {"$type": "Data", "query": "fo{o.b}ar"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithDataTypeResolver{
					DataTypeResolver: resolve.NewMemoryDataTypeResolver(
						map[string]interface{}{"foo": map[string]string{"bar": "baz"}},
					),
				},
			},
			ExpectedError: &model.PathEvaluationError{
				Path: "data",
				Cause: &evaluate.InvalidTypeError{
					Type: "Data",
					Cause: &evaluate.DataQueryError{
						Query: "fo{o.b}ar",
						Cause: fmt.Errorf("parsing error: fo{o.b}ar\t:1:3 - 1:4 unexpected \"{\" while scanning operator"),
					},
				},
			},
		},
		{
			Name: "unresolvable invocation",
			Data: `{"foo": {"$fn.foo": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONInvocation("foo", "bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{
					{Name: "foo", Cause: fn.ErrFunctionNotFound},
				},
			},
		},
		{
			Name: "unresolvable invocation in template",
			Data: `{"foo": "${foo('bar')}"}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${foo('bar')}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{
					{Name: "foo", Cause: fn.ErrFunctionNotFound},
				},
			},
		},
		{
			Name: "many unresolvable",
			Data: `{
				"a": {"$type": "Secret", "name": "foo"},
				"b": {"$type": "Output", "from": "baz", "name": "bar"},
				"c": {"$type": "Parameter", "name": "quux"},
				"d": {"$fn.foo": "bar"},
				"e": "hello",
				"f": {"$type": "Answer", "askRef": "baz", "name": "bar"},
				"g": {"$type": "Connection", "type": "blort", "name": "bar"}
			}`,
			ExpectedValue: map[string]interface{}{
				"a": testutil.JSONSecret("foo"),
				"b": testutil.JSONOutput("baz", "bar"),
				"c": testutil.JSONParameter("quux"),
				"d": testutil.JSONInvocation("foo", "bar"),
				"e": "hello",
				"f": testutil.JSONAnswer("baz", "bar"),
				"g": testutil.JSONConnection("blort", "bar"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "foo"},
				},
				Outputs: []model.UnresolvableOutput{
					{From: "baz", Name: "bar"},
				},
				Parameters: []model.UnresolvableParameter{
					{Name: "quux"},
				},
				Invocations: []model.UnresolvableInvocation{
					{Name: "foo", Cause: fn.ErrFunctionNotFound},
				},
				Answers: []model.UnresolvableAnswer{
					{AskRef: "baz", Name: "bar"},
				},
				Connections: []model.UnresolvableConnection{
					{Type: "blort", Name: "bar"},
				},
			},
		},
		{
			Name: "many unresolvable in template",
			Data: `{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "${foo('bar')}",
				"e": "hello",
				"g": "${connections.blort.bar}"
			}`,
			ExpectedValue: map[string]interface{}{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "${foo('bar')}",
				"e": "hello",
				"g": "${connections.blort.bar}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "foo"},
				},
				Outputs: []model.UnresolvableOutput{
					{From: "baz", Name: "bar"},
				},
				Parameters: []model.UnresolvableParameter{
					{Name: "quux"},
				},
				Invocations: []model.UnresolvableInvocation{
					{Name: "foo", Cause: fn.ErrFunctionNotFound},
				},
				Connections: []model.UnresolvableConnection{
					{Type: "blort", Name: "bar"},
				},
			},
		},
		{
			Name: "unresolvable at depth",
			Data: `{
				"foo": [
					{"a": {"$type": "Secret", "name": "foo"}},
					{"$type": "Parameter", "name": "bar"}
				],
				"bar": {"$type": "Parameter", "name": "frob"}
			}`,
			Depth: 3,
			ExpectedValue: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{"a": testutil.JSONSecret("foo")},
					testutil.JSONParameter("bar"),
				},
				"bar": testutil.JSONParameter("frob"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "bar"},
					{Name: "frob"},
				},
			},
		},
		{
			Name: "unresolvable at depth in template",
			Data: `{
				"foo": [
					{"a": "${secrets.foo}"},
					"${parameters.bar}"
				],
				"bar": "${parameters.frob}"
			}`,
			Depth: 3,
			ExpectedValue: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{"a": "${secrets.foo}"},
					"${parameters.bar}",
				},
				"bar": "${parameters.frob}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "bar"},
					{Name: "frob"},
				},
			},
		},
		{
			Name: "resolvable",
			Data: `{
				"a": {"$type": "Secret", "name": "foo"},
				"b": {"$type": "Output", "from": "baz", "name": "bar"},
				"c": {"$type": "Parameter", "name": "quux"},
				"d": {"$fn.foo": "bar"},
				"e": "hello",
				"f": {"$type": "Answer", "askRef": "baz", "name": "bar"},
				"g": {"$type": "Data", "query": "foo.bar"},
				"h": {"$type": "Connection", "type": "blort", "name": "bar"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"quux": []interface{}{1, 2, 3}},
					),
				},
				evaluate.WithFunctionMap{Map: fns},
				evaluate.WithAnswerTypeResolver{
					AnswerTypeResolver: resolve.NewMemoryAnswerTypeResolver(
						map[resolve.MemoryAnswerKey]interface{}{
							{AskRef: "baz", Name: "bar"}: "approved",
						},
					),
				},
				evaluate.WithDataTypeResolver{
					Name:    "event",
					Default: true,
					DataTypeResolver: resolve.NewMemoryDataTypeResolver(
						map[string]interface{}{"foo": map[string]interface{}{"bar": "baz"}},
					),
				},
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "blort", Name: "bar"}: map[string]interface{}{"bar": "blort"},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"a": "v3ry s3kr3t!",
				"b": "127.0.0.1",
				"c": []interface{}{1, 2, 3},
				"d": "~~[bar]~~",
				"e": "hello",
				"f": "approved",
				"g": "baz",
				"h": map[string]interface{}{"bar": "blort"},
			},
		},
		{
			Name: "resolvable in template",
			Data: `{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "${foo('bar')}",
				"e": "hello",
				"g": "${event.foo.bar}",
				"h": "${connections.blort.bar}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"quux": []interface{}{1, 2, 3}},
					),
				},
				evaluate.WithFunctionMap{Map: fns},
				evaluate.WithDataTypeResolver{
					Name: "event",
					DataTypeResolver: resolve.NewMemoryDataTypeResolver(
						map[string]interface{}{"foo": map[string]interface{}{"bar": "baz"}},
					),
				},
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "blort", Name: "bar"}: map[string]interface{}{"bar": "blort"},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"a": "v3ry s3kr3t!",
				"b": "127.0.0.1",
				"c": []interface{}{1, 2, 3},
				"d": "~~[bar]~~",
				"e": "hello",
				"g": "baz",
				"h": map[string]interface{}{"bar": "blort"},
			},
		},
		{
			Name: "resolvable expansion in template",
			Data: `{
				"foo": "${$}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"quux": []interface{}{1, 2, 3}},
					),
				},
				evaluate.WithFunctionMap{Map: fns},
				evaluate.WithDataTypeResolver{
					Name: "event",
					DataTypeResolver: resolve.NewMemoryDataTypeResolver(
						map[string]interface{}{"foo": map[string]interface{}{"bar": "baz"}},
					),
				},
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "blort", Name: "bar"}:  map[string]interface{}{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]interface{}{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]interface{}{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"secrets": map[string]interface{}{
						"foo": "v3ry s3kr3t!",
					},
					"outputs": map[string]interface{}{
						"baz": map[string]interface{}{
							"bar": "127.0.0.1",
						},
					},
					"parameters": map[string]interface{}{
						"quux": []interface{}{1, 2, 3},
					},
					"event": map[string]interface{}{
						"foo": map[string]interface{}{
							"bar": "baz",
						},
					},
					"connections": map[string]interface{}{
						"blort": map[string]interface{}{
							"bar":  map[string]interface{}{"bar": "blort"},
							"wish": map[string]interface{}{"sim": "jax"},
						},
						"zup": map[string]interface{}{
							"bar": map[string]interface{}{"waz": "mux"},
						},
					},
				},
			},
		},
		{
			Name: "nested resolvable",
			Data: `{
				"aws": {
					"accessKeyID": {"$type": "Secret", "name": "accessKeyID"},
					"secretAccessKey": {"$type": "Secret", "name": "secretAccessKey"}
				},
				"instanceID": {"$type": "Parameter", "name": "instanceID"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"accessKeyID": "AKIANOAHISCOOL", "secretAccessKey": "abcdefs3cr37s"},
					),
				},
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"instanceID": "i-abcdef123456"},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "AKIANOAHISCOOL",
					"secretAccessKey": "abcdefs3cr37s",
				},
				"instanceID": "i-abcdef123456",
			},
		},
		{
			Name: "nested resolvable in template",
			Data: `{
				"aws": {
					"accessKeyID": "${secrets.accessKeyID}",
					"secretAccessKey": "${secrets.secretAccessKey}"
				},
				"instanceID": "${parameters.instanceID}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"accessKeyID": "AKIANOAHISCOOL", "secretAccessKey": "abcdefs3cr37s"},
					),
				},
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"instanceID": "i-abcdef123456"},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "AKIANOAHISCOOL",
					"secretAccessKey": "abcdefs3cr37s",
				},
				"instanceID": "i-abcdef123456",
			},
		},
		{
			Name: "resolvable parameter traversal",
			Data: `{
				"accessKeyID": "${parameters.aws.accessKeyID}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"aws": map[string]interface{}{"accessKeyID": "foo", "secretAccessKey": "bar"}},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"accessKeyID": "foo",
			},
		},
		{
			Name: "resolvable output traversal",
			Data: `{
				"test": "${outputs.baz.bar.b[1]}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: map[string]interface{}{
								"a": "test",
								"b": []interface{}{"c", "d"},
							},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"test": "d",
			},
		},
		{
			Name: "resolvable parameter in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
		},
		{
			Name: "resolvable parameter in invocation argument in partial template",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": "${parameters.aws}"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
		},
		{
			Name: "resolvable parameter in invocation argument in template",
			Data: `{
				"aws": "${jsonUnmarshal(parameters.aws)}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
		},
		{
			Name: "unresolvable parameter in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			ExpectedValue: map[string]interface{}{
				"aws": testutil.JSONInvocation("jsonUnmarshal", []interface{}{testutil.JSONParameter("aws")}),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "aws"},
				},
			},
		},
		{
			Name: "unresolvable parameter in invocation argument in partial template",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": "${parameters.aws}"}
			}`,
			ExpectedValue: map[string]interface{}{
				"aws": testutil.JSONInvocation("jsonUnmarshal", []interface{}{"${parameters.aws}"}),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "aws"},
				},
			},
		},
		{
			Name: "unresolvable parameter in invocation argument in template",
			Data: `{
				"aws": "${jsonUnmarshal(parameters.aws)}"
			}`,
			ExpectedValue: map[string]interface{}{
				"aws": "${jsonUnmarshal(parameters.aws)}",
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "aws"},
				},
			},
		},
		{
			Name: "partially resolvable invocation",
			Data: `{
				"foo": {
					"$fn.concat": [
						{"$type": "Parameter", "name": "first"},
						{"$type": "Parameter", "name": "second"}
					]
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONInvocation("concat", []interface{}{
					"bar",
					testutil.JSONParameter("second"),
				}),
			},
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"first": "bar"},
					),
				},
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "second"},
				},
			},
		},
		{
			Name: "partially resolvable invocation in partial template",
			Data: `{
				"foo": {
					"$fn.concat": [
						"${parameters.first}",
						"${parameters.second}"
					]
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONInvocation("concat", []interface{}{
					"bar",
					"${parameters.second}",
				}),
			},
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"first": "bar"},
					),
				},
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "second"},
				},
			},
		},
		{
			Name: "partially resolvable invocation in template",
			Data: `{
				"foo": "${concat(parameters.first, parameters.second)}"
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": "${concat(parameters.first, parameters.second)}",
			},
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"first": "bar"},
					),
				},
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "second"},
				},
			},
		},
		{
			Name: "successful invocation of fn.convertMarkdown to Jira syntax",
			Data: `{
				"foo": {
					"$fn.convertMarkdown": [
						"jira",` +
				"\"--- `code` ---\"" + `
					]
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": "\n----\n{code}code{code}\n----\n",
			},
		},
		{
			Name: "successful invocation of fn.convertMarkdown to Jira syntax in template",
			Data: fmt.Sprintf(`{"foo": %q}`, "${convertMarkdown('jira', '--- `code` ---')}"),
			ExpectedValue: map[string]interface{}{
				"foo": "\n----\n{code}code{code}\n----\n",
			},
		},
		{
			Name: "unresolved conditionals evaluation",
			Data: `{
				"conditions": [{"$fn.equals": [
					{"$type": "Parameter", "name": "first"},
					"foobar"
				]}]
			}`,
			ExpectedValue: map[string]interface{}{
				"conditions": []interface{}{testutil.JSONInvocation("equals", []interface{}{
					testutil.JSONParameter("first"),
					"foobar",
				})},
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "first"},
				},
			},
		},
		{
			Name: "unresolved conditionals evaluation in template",
			Data: `{
				"conditions": ["${parameters.first == 'foobar'}"]
			}`,
			ExpectedValue: map[string]interface{}{
				"conditions": []interface{}{
					"${parameters.first == 'foobar'}",
				},
			},
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "first"},
				},
			},
		},
		{
			Name: "resolved conditionals evaluation",
			Data: `{
				"conditions": [
					{"$fn.equals": [
						{"$type": "Parameter", "name": "first"},
						"foobar"
					]},
					{"$fn.notEquals": [
						{"$type": "Parameter", "name": "first"},
						"barfoo"
					]}
				]
			}`,
			ExpectedValue: map[string]interface{}{
				"conditions": []interface{}{true, true},
			},
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"first": "foobar"},
					),
				},
			},
		},
		{
			Name: "resolved conditionals evaluation in template",
			Data: `{
				"conditions": [
					"${parameters.first == 'foobar'}",
					"${parameters.first != 'barfoo'}"
				]
			}`,
			ExpectedValue: map[string]interface{}{
				"conditions": []interface{}{true, true},
			},
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{"first": "foobar"},
					),
				},
			},
		},
		{
			Name: "encoded string",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "SGVsbG8sIJCiikU="
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": "Hello, \x90\xA2\x8A\x45",
			},
		},
		{
			Name: "encoded string from secret",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": {"$type": "Secret", "name": "bar"}
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"bar": "SGVsbG8sIJCiikU="},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": "Hello, \x90\xA2\x8A\x45",
			},
		},
		{
			Name: "encoded string from secret in template",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "${secrets.bar}"
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"bar": "SGVsbG8sIJCiikU="},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": "Hello, \x90\xA2\x8A\x45",
			},
		},
		{
			Name: "encoded string from unresolvable secret",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": {"$type": "Secret", "name": "bar"}
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONEncoding(transfer.Base64EncodingType, testutil.JSONSecret("bar")),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "encoded string from unresolvable secret in template",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "${secrets.bar}"
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONEncoding(transfer.Base64EncodingType, "${secrets.bar}"),
			},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "bar"},
				},
			},
		},
		{
			Name:          "invocation with array arguments",
			Data:          `{"$fn.foo": ["bar", "baz"]}`,
			Opts:          []evaluate.Option{evaluate.WithFunctionMap{Map: fns}},
			ExpectedValue: "~~[bar baz]~~",
		},
		{
			Name:          "invocation with array arguments in template",
			Data:          `"${foo('bar', 'baz')}"`,
			Opts:          []evaluate.Option{evaluate.WithFunctionMap{Map: fns}},
			ExpectedValue: "~~[bar baz]~~",
		},
		{
			Name:          "invocation with object arguments",
			Data:          `{"$fn.foo": {"whiz": "bang", "not": "this"}}`,
			Opts:          []evaluate.Option{evaluate.WithFunctionMap{Map: fns}},
			ExpectedValue: "~~bang~~",
		},
		{
			Name:          "invocation with object arguments in template",
			Data:          `"${foo(whiz: 'bang', not: 'this')}"`,
			Opts:          []evaluate.Option{evaluate.WithFunctionMap{Map: fns}},
			ExpectedValue: "~~bang~~",
		},
		{
			Name: "bad invocation",
			Data: `{"$fn.append": [1, 2, 3]}`,
			ExpectedError: &model.InvocationError{
				Name: "append",
				Cause: &fn.PositionalArgError{
					Arg: 1,
					Cause: &fn.UnexpectedTypeError{
						Wanted: []reflect.Type{reflect.TypeOf([]interface{}(nil))},
						Got:    reflect.TypeOf(float64(0)),
					},
				},
			},
		},
		{
			Name: "bad invocation in template",
			Data: `"${append(1, 2, 3)}"`,
			ExpectedError: &template.EvaluationError{
				Start: "${",
				Cause: &model.InvocationError{
					Name: "append",
					Cause: &fn.PositionalArgError{
						Arg: 1,
						Cause: &fn.UnexpectedTypeError{
							Wanted: []reflect.Type{reflect.TypeOf([]interface{}(nil))},
							Got:    reflect.TypeOf(float64(0)),
						},
					},
				},
			},
		},
		{
			Name: "resolvable template dereferencing",
			Data: `"${secrets['regions.' + parameters.region]}"`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"region": "us-east-1",
						},
					),
				},
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: "EAST",
		},
		{
			Name: "unresolvable template dereferencing",
			Data: `"${secrets['regions.' + parameters.region]}"`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"region": "us-east-2",
						},
					),
				},
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: `${secrets['regions.' + parameters.region]}`,
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "regions.us-east-2"},
				},
			},
		},
		{
			Name:          "nested unresolvable template dereferencing",
			Data:          `"${secrets['regions.' + parameters.region]}"`,
			ExpectedValue: `${secrets['regions.' + parameters.region]}`,
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "region"},
				},
			},
		},
		{
			Name: "parameters expansion in template",
			Data: `{
				"foo": "${parameters}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"region": "us-east-1",
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"region": "us-east-1",
				},
			},
		},
		{
			Name: "secrets expansion in template",
			Data: `{
				"foo": "${secrets}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"regions.us-east-1": "EAST",
					"regions.us-west-1": "WEST",
				},
			},
		},
		{
			Name: "outputs expansion in template",
			Data: `{
				"foo": "${outputs}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"baz": map[string]interface{}{
						"bar": "127.0.0.1",
					},
				},
			},
		},
		{
			Name: "single step output expansion in template",
			Data: `{
				"foo": "${outputs.baz}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "127.0.0.1",
				},
			},
		},
		{
			Name: "connections expansion in template",
			Data: `{
				"foo": "${connections}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "blort", Name: "bar"}:  map[string]interface{}{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]interface{}{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]interface{}{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"blort": map[string]interface{}{
						"bar":  map[string]interface{}{"bar": "blort"},
						"wish": map[string]interface{}{"sim": "jax"},
					},
					"zup": map[string]interface{}{
						"bar": map[string]interface{}{"waz": "mux"},
					},
				},
			},
		},
		{
			Name: "single connection type expansion in template",
			Data: `{
				"foo": "${connections.blort}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "blort", Name: "bar"}:  map[string]interface{}{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]interface{}{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]interface{}{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar":  map[string]interface{}{"bar": "blort"},
					"wish": map[string]interface{}{"sim": "jax"},
				},
			},
		},
		{
			Name: "template interpolation",
			Data: `{
				"foo": "Hello, ${secrets.who}!"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"who": "friend"},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": "Hello, friend!",
			},
		},
		{
			Name: "template interpolation with mapping type",
			Data: `{
				"foo": "Some secret people:\n${secrets}"
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"who": "friend"},
					),
				},
			},
			ExpectedValue: map[string]interface{}{
				"foo": `Some secret people:
{
	"who": "friend"
}`,
			},
		},
	}.RunAll(t)
}

func TestEvaluateQuery(t *testing.T) {
	tests{
		{
			Name:          "literal",
			Data:          `{"foo": "bar"}`,
			Query:         `foo`,
			ExpectedValue: "bar",
		},
		{
			Name:  "nonexistent key",
			Data:  `{"foo": [{"bar": "baz"}]}`,
			Query: `foo[0].quux`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "0",
					Cause: &model.PathEvaluationError{
						Path:  "quux",
						Cause: &eval.UnknownKeyError{Key: "quux"},
					},
				},
			},
		},
		{
			Name:  "nonexistent index",
			Data:  `{"foo": [{"bar": "baz"}]}`,
			Query: `foo[1].quux`,
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path:  "1",
					Cause: &eval.IndexOutOfBoundsError{Index: 1},
				},
			},
		},
		{
			Name: "traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"bar": map[string]interface{}{
								"bar": map[string]interface{}{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
		},
		{
			Name: "JSONPath traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.foo.bar.baz",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"bar": map[string]interface{}{
								"bar": map[string]interface{}{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
		},
		{
			Name: "JSONPath traverses output",
			Data: `{
				"foo": {"$type": "Output", "from": "baz", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.foo.b[1]",
			Opts: []evaluate.Option{
				evaluate.WithOutputTypeResolver{
					OutputTypeResolver: resolve.NewMemoryOutputTypeResolver(
						map[resolve.MemoryOutputKey]interface{}{
							{From: "baz", Name: "bar"}: map[string]interface{}{
								"a": "test",
								"b": []interface{}{"c", "d"},
							},
						},
					),
				},
			},
			ExpectedValue: "d",
		},
		{
			Name: "JSONPath template traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage,
			Query:         "{.foo.bar.baz}",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(
						map[string]interface{}{
							"bar": map[string]interface{}{
								"bar": map[string]interface{}{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
		},
		{
			Name: "unresolvable",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "JSONPath unresolvable",
			Data: `{
				"a": {"name": "aa", "value": {"$type": "Secret", "name": "foo"}},
				"b": {"name": "bb", "value": {"$type": "Secret", "name": "bar"}},
				"c": {"name": "cc", "value": "gggggg"}
			}`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.*.value",
			ExpectedValue: []interface{}{"gggggg"},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "bar"},
					{Name: "foo"},
				},
			},
		},
		{
			Name: "unresolvable not evaluated because not in path",
			Data: `{
				"a": {"$type": "Parameter", "name": "bar"},
				"b": {"c": {"$type": "Secret", "name": "foo"}}
			}`,
			Query: "b.c",
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"foo": "very secret"},
					),
				},
			},
			ExpectedValue: "very secret",
		},
		{
			Name: "JSONPath object unresolvable not evaluated because not in path",
			Data: `{
				"a": {"name": "aa", "value": {"$type": "Parameter", "name": "bar"}},
				"b": {"name": "bb", "value": {"$type": "Secret", "name": "foo"}}
			}`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.*.name",
			ExpectedValue: randomOrder{"aa", "bb"},
		},
		{
			Name: "JSONPath array unresolvable not evaluated because not in path",
			Data: `[
				{"name": "aa", "value": {"$type": "Parameter", "name": "bar"}},
				{"name": "bb", "value": {"$type": "Secret", "name": "foo"}}
			]`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.*.name",
			ExpectedValue: randomOrder{"aa", "bb"},
		},
		{
			Name: "type resolver returns an unsupported type",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"}
			}`,
			Query: "a.inner",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
						"foo": map[string]string{"inner": "test"},
					}),
				},
			},
			ExpectedError: &model.PathEvaluationError{
				Path: "a",
				Cause: &model.PathEvaluationError{
					Path: "inner",
					Cause: &model.UnsupportedValueError{
						Type: reflect.TypeOf(map[string]string(nil)),
					},
				},
			},
		},
		{
			Name: "type resolver returns an unsupported type in JSONPath",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"},
				"b": {"$type": "Parameter", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathLanguage,
			Query:         "$.a.inner",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
						"foo": map[string]string{"inner": "test"},
						"bar": map[string]interface{}{"inner": "test"},
					}),
				},
			},
			ExpectedError: &model.UnsupportedValueError{
				Type: reflect.TypeOf(map[string]string(nil)),
			},
		},
		{
			Name: "type resolver returns an unsupported type in JSONPath template",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"}
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage,
			Query:         "{.a.inner}",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
						"foo": map[string]string{"inner": "test"},
					}),
				},
			},
			ExpectedError: &template.EvaluationError{
				Start: "{",
				Cause: &model.UnsupportedValueError{
					Type: reflect.TypeOf(map[string]string(nil)),
				},
			},
		},
		{
			Name: "JSONPath template traverses object",
			Data: `{
				"args": {
					"a": "undo",
					"b": {"$fn.concat": ["deployment.v1.apps/", {"$type": "Parameter", "name": "deployment"}]}
				}
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage,
			Query:         "{.args}",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
						"deployment": "my-test-deployment",
					}),
				},
			},
			ExpectedValue: "map[a:undo b:deployment.v1.apps/my-test-deployment]",
		},
		{
			Name: "JSONPath template traverses array",
			Data: `{
				"args": [
					"undo",
					{"$fn.concat": ["deployment.v1.apps/", {"$type": "Parameter", "name": "deployment"}]}
				]
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage,
			Query:         "{.args}",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver{
					ParameterTypeResolver: resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
						"deployment": "my-test-deployment",
					}),
				},
			},
			ExpectedValue: "undo deployment.v1.apps/my-test-deployment",
		},
		{
			Name: "JSONPath template traverses array with unresolvables",
			Data: `{
				"args": [
					"undo",
					{"$fn.concat": ["deployment.v1.apps/", {"$type": "Parameter", "name": "deployment"}]}
				]
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage,
			Query:         "{.args}",
			ExpectedValue: "undo map[$fn.concat:[deployment.v1.apps/ map[$type:Parameter name:deployment]]]",
			ExpectedUnresolvable: model.Unresolvable{
				Parameters: []model.UnresolvableParameter{
					{Name: "deployment"},
				},
			},
		},
		{
			Name:  "query has an error under a path",
			Data:  `{"foo": {"bar": ["baz", "quux"]}}`,
			Query: "foo.bar[0].nope",
			ExpectedError: &model.PathEvaluationError{
				Path: "foo",
				Cause: &model.PathEvaluationError{
					Path: "bar",
					Cause: &model.PathEvaluationError{
						Path: "0",
						Cause: &model.PathEvaluationError{
							Path: "nope",
							Cause: &eval.UnsupportedValueTypeError{
								Value: "baz",
								Field: "nope",
							},
						},
					},
				},
			},
		},
	}.RunAll(t)
}

func TestEvaluateIntoBasic(t *testing.T) {
	type foo struct {
		Bar string `spec:"bar"`
	}

	type root struct {
		Foo foo `spec:"foo"`
	}

	tests{
		{
			Name:          "basic",
			Data:          `{"foo": {"bar": "baz"}}`,
			Into:          &root{},
			ExpectedValue: &root{Foo: foo{Bar: "baz"}},
		},
		{
			Name: "resolvable",
			Data: `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"foo": "v3ry s3kr3t!"},
					),
				},
			},
			Into:          &root{},
			ExpectedValue: &root{Foo: foo{Bar: "v3ry s3kr3t!"}},
		},
		{
			Name:          "unresolvable",
			Data:          `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Into:          &root{Foo: foo{Bar: "masked"}},
			ExpectedValue: &root{},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "foo"},
				},
			},
		},
		{
			Name: "map",
			Data: `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"foo": "v3ry s3kr3t!"},
					),
				},
			},
			Into:          &map[string]interface{}{},
			ExpectedValue: &map[string]interface{}{"foo": map[string]interface{}{"bar": "v3ry s3kr3t!"}},
		},
	}.RunAll(t)
}

func TestEvaluatePath(t *testing.T) {
	type awsDetails struct {
		AccessKeyID     string
		SecretAccessKey string
		Region          string
	}

	type awsSpec struct {
		AWS *awsDetails
	}

	tests{
		{
			Name: "resolvable (using connections)",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithConnectionTypeResolver{
					ConnectionTypeResolver: resolve.NewMemoryConnectionTypeResolver(
						map[resolve.MemoryConnectionKey]interface{}{
							{Type: "aws", Name: "aws"}: map[string]interface{}{
								"accessKeyID":     "AKIANOAHISCOOL",
								"secretAccessKey": "abcdefs3cr37s",
							},
						},
					),
				},
			},
			Into: &awsSpec{},
			ExpectedValue: &awsSpec{
				AWS: &awsDetails{
					AccessKeyID:     "AKIANOAHISCOOL",
					SecretAccessKey: "abcdefs3cr37s",
					Region:          "us-west-2",
				},
			},
		},
		{
			Name: "unresolvable (using connections)",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			ExpectedValue: map[string]interface{}(
				map[string]interface{}{
					"aws": map[string]interface{}{
						"accessKeyID": map[string]interface{}{
							"$fn.path": map[string]interface{}{
								"object": map[string]interface{}{
									"$type": "Connection", "name": "aws", "type": "aws"},
								"query": "accessKeyID"}},
						"secretAccessKey": map[string]interface{}{
							"$fn.path": map[string]interface{}{
								"object": map[string]interface{}{
									"$type": "Connection", "name": "aws", "type": "aws"},
								"query": "secretAccessKey"}},
						"region": "us-west-2",
					},
				}),
			ExpectedUnresolvable: model.Unresolvable{
				Connections: []model.UnresolvableConnection{
					{Type: "aws", Name: "aws"},
				},
			},
		},
		{
			Name: "resolvable (using secrets)",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{"aws": `{
							"accessKeyID": "AKIANOAHISCOOL",
							"secretAccessKey": "abcdefs3cr37s"
						}`,
						},
					),
				},
			},
			Into: &awsSpec{},
			ExpectedValue: &awsSpec{
				AWS: &awsDetails{
					AccessKeyID:     "AKIANOAHISCOOL",
					SecretAccessKey: "abcdefs3cr37s",
					Region:          "us-west-2",
				},
			},
		},
		{
			Name: "unresolvable (using secrets)",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID": map[string]interface{}{
						"$fn.path": map[string]interface{}{
							"object": map[string]interface{}{
								"$fn.jsonUnmarshal": []interface{}{map[string]interface{}{
									"$type": "Secret", "name": "aws"}}},
							"query": "accessKeyID"}},
					"secretAccessKey": map[string]interface{}{
						"$fn.path": map[string]interface{}{
							"object": map[string]interface{}{
								"$fn.jsonUnmarshal": []interface{}{map[string]interface{}{
									"$type": "Secret", "name": "aws"}}},
							"query": "secretAccessKey"}},
					"region": "us-west-2",
				}},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "aws"},
				},
			},
		},
	}.RunAll(t)
}

func TestEvaluateIntoStepHelper(t *testing.T) {
	type awsDetails struct {
		AccessKeyID     string
		SecretAccessKey string
		Region          string
	}

	type awsSpec struct {
		AWS *awsDetails
	}

	tests{
		{
			Name: "unresolvable",
			Data: `{
				"aws": {
					"accessKeyID": {"$type": "Secret", "name": "aws.accessKeyID"},
					"secretAccessKey": {"$type": "Secret", "name": "aws.secretAccessKey"},
					"region": "us-west-2"
				},
				"op": {"$type": "Parameter", "name": "op"}
			}`,
			Into:          &awsSpec{},
			ExpectedValue: &awsSpec{AWS: &awsDetails{Region: "us-west-2"}},
			ExpectedUnresolvable: model.Unresolvable{
				Secrets: []model.UnresolvableSecret{
					{Name: "aws.accessKeyID"},
					{Name: "aws.secretAccessKey"},
				},
			},
		},
		{
			Name: "resolvable",
			Data: `{
				"aws": {
					"accessKeyID": {"$type": "Secret", "name": "aws.accessKeyID"},
					"secretAccessKey": {"$type": "Secret", "name": "aws.secretAccessKey"},
					"region": "us-west-2"
				},
				"op": {"$type": "Parameter", "name": "op"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{
							"aws.accessKeyID":     "AKIANOAHISCOOL",
							"aws.secretAccessKey": "abcdefs3cr37s",
						},
					),
				},
			},
			Into: &awsSpec{},
			ExpectedValue: &awsSpec{
				AWS: &awsDetails{
					AccessKeyID:     "AKIANOAHISCOOL",
					SecretAccessKey: "abcdefs3cr37s",
					Region:          "us-west-2",
				},
			},
		},
		{
			Name: "resolvable traverses",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Secret", "name": "aws"}},
				"op": {"$type": "Parameter", "name": "op"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver{
					SecretTypeResolver: resolve.NewMemorySecretTypeResolver(
						map[string]string{
							"aws": `{
							"accessKeyID": "AKIANOAHISCOOL",
							"secretAccessKey": "abcdefs3cr37s",
							"region": "us-west-2",
							"extra": "unused"
						}`,
						},
					),
				},
			},
			Into: &awsSpec{},
			ExpectedValue: &awsSpec{
				AWS: &awsDetails{
					AccessKeyID:     "AKIANOAHISCOOL",
					SecretAccessKey: "abcdefs3cr37s",
					Region:          "us-west-2",
				},
			},
		},
	}.RunAll(t)
}
