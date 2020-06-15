package evaluate_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/evaluate"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/fn"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/testutil"
	"github.com/stretchr/testify/require"
)

type randomOrder []interface{}

type test struct {
	Name                 string
	Data                 string
	Opts                 []evaluate.Option
	Depth                int
	Query                string
	Into                 interface{}
	ExpectedValue        interface{}
	ExpectedUnresolvable evaluate.Unresolvable
	ExpectedError        error
}

func (tt test) Run(t *testing.T) {
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
	var u evaluate.Unresolvable
	if tt.Query != "" {
		r, err := ev.EvaluateQuery(context.Background(), tree, tt.Query)
		check(t, err)

		if r != nil {
			v = r.Value
			u = r.Unresolvable
		}
	} else if tt.Into != nil {
		u, err = ev.EvaluateInto(context.Background(), tree, tt.Into)
		check(t, err)

		v = tt.Into
	} else {
		depth := tt.Depth
		if depth == 0 {
			depth = -1
		}

		r, err := ev.Evaluate(context.Background(), tree, depth)
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
	fns := resolve.NewMemoryInvocationResolver(
		fn.NewMap(map[string]fn.Descriptor{
			"foo": fn.DescriptorFuncs{
				PositionalInvokerFunc: func(args []interface{}) (fn.Invoker, error) {
					return fn.InvokerFunc(func(ctx context.Context) (interface{}, error) {
						return fmt.Sprintf("~~%v~~", args), nil
					}), nil
				},
				KeywordInvokerFunc: func(args map[string]interface{}) (fn.Invoker, error) {
					return fn.InvokerFunc(func(ctx context.Context) (interface{}, error) {
						return fmt.Sprintf("~~%v~~", args["whiz"]), nil
					}), nil
				},
			},
		}),
	)

	tests{
		{
			Name:          "literal",
			Data:          `{"foo": "bar"}`,
			ExpectedValue: map[string]interface{}{"foo": "bar"},
		},
		{
			Name: "unresolvable data",
			Data: `{"baz": {"$type": "Data", "query": "foo.bar"}}`,
			ExpectedValue: map[string]interface{}{
				"baz": testutil.JSONData("foo.bar"),
			},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Data: []evaluate.UnresolvableData{
					{Query: "foo.bar"},
				},
			},
		},
		{
			Name: "unresolvable secret",
			Data: `{"foo": {"$type": "Secret", "name": "bar"}}`,
			ExpectedValue: map[string]interface{}{
				"foo": testutil.JSONSecret("bar"),
			},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Connections: []evaluate.UnresolvableConnection{
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Outputs: []evaluate.UnresolvableOutput{
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
					{Name: "bar"},
				},
			},
		},
		{
			Name: "invalid data",
			Data: `{"foo": [{"$type": "Data"}]}`,
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
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
				evaluate.WithDataTypeResolver(resolve.NewMemoryDataTypeResolver(
					map[string]interface{}{"foo": map[string]string{"bar": "baz"}},
				)),
			},
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "data",
				Cause: &evaluate.InvalidTypeError{
					Type: "Data",
					Cause: &resolve.DataQueryError{
						Query: "fo{o.b}ar",
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Invocations: []evaluate.UnresolvableInvocation{
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
				"g": {"$type": "Data", "query": "foo.bar"},
				"h": {"$type": "Connection", "type": "blort", "name": "bar"}
			}`,
			ExpectedValue: map[string]interface{}{
				"a": testutil.JSONSecret("foo"),
				"b": testutil.JSONOutput("baz", "bar"),
				"c": testutil.JSONParameter("quux"),
				"d": testutil.JSONInvocation("foo", "bar"),
				"e": "hello",
				"f": testutil.JSONAnswer("baz", "bar"),
				"g": testutil.JSONData("foo.bar"),
				"h": testutil.JSONConnection("blort", "bar"),
			},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
					{Name: "foo"},
				},
				Outputs: []evaluate.UnresolvableOutput{
					{From: "baz", Name: "bar"},
				},
				Parameters: []evaluate.UnresolvableParameter{
					{Name: "quux"},
				},
				Invocations: []evaluate.UnresolvableInvocation{
					{Name: "foo", Cause: fn.ErrFunctionNotFound},
				},
				Answers: []evaluate.UnresolvableAnswer{
					{AskRef: "baz", Name: "bar"},
				},
				Data: []evaluate.UnresolvableData{
					{Query: "foo.bar"},
				},
				Connections: []evaluate.UnresolvableConnection{
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"foo": "v3ry s3kr3t!"},
				)),
				evaluate.WithOutputTypeResolver(resolve.NewMemoryOutputTypeResolver(
					map[resolve.MemoryOutputKey]interface{}{
						{From: "baz", Name: "bar"}: "127.0.0.1",
					},
				)),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"quux": []interface{}{1, 2, 3}},
				)),
				evaluate.WithInvocationResolver(fns),
				evaluate.WithAnswerTypeResolver(resolve.NewMemoryAnswerTypeResolver(
					map[resolve.MemoryAnswerKey]interface{}{
						{AskRef: "baz", Name: "bar"}: "approved",
					},
				)),
				evaluate.WithDataTypeResolver(resolve.NewMemoryDataTypeResolver(
					map[string]interface{}{"foo": map[string]string{"bar": "baz"}},
				)),
				evaluate.WithConnectionTypeResolver(resolve.NewMemoryConnectionTypeResolver(
					map[resolve.MemoryConnectionKey]interface{}{
						{Type: "blort", Name: "bar"}: map[string]interface{}{"bar": "blort"},
					},
				)),
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
			Name: "nested resolvable",
			Data: `{
				"aws": {
					"accessKeyID": {"$type": "Secret", "name": "accessKeyID"},
					"secretAccessKey": {"$type": "Secret", "name": "secretAccessKey"}
				},
				"instanceID": {"$type": "Parameter", "name": "instanceID"}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"accessKeyID": "AKIANOAHISCOOL", "secretAccessKey": "abcdefs3cr37s"},
				)),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"instanceID": "i-abcdef123456"},
				)),
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
			Name: "resolvable secret in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
				)),
			},
			ExpectedValue: map[string]interface{}{
				"aws": map[string]interface{}{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
		},
		{
			Name: "unresolvable secret in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			ExpectedValue: map[string]interface{}{
				"aws": testutil.JSONInvocation("jsonUnmarshal", testutil.JSONParameter("aws")),
			},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
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
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"first": "bar"},
				)),
			},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
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
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"first": "foobar"},
				)),
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"bar": "SGVsbG8sIJCiikU="},
				)),
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
					{Name: "bar"},
				},
			},
		},
		{
			Name:          "invocation with array arguments",
			Data:          `{"$fn.foo": ["bar", "baz"]}`,
			Opts:          []evaluate.Option{evaluate.WithInvocationResolver(fns)},
			ExpectedValue: "~~[bar baz]~~",
		},
		{
			Name:          "invocation with object arguments",
			Data:          `{"$fn.foo": {"whiz": "bang", "not": "this"}}`,
			Opts:          []evaluate.Option{evaluate.WithInvocationResolver(fns)},
			ExpectedValue: "~~bang~~",
		},
		{
			Name: "custom invocation",
			Data: `{"$fn.foo": {"whiz": "bang", "not": "this"}}`,
			Opts: []evaluate.Option{
				evaluate.WithInvocationResolver(fns),
				evaluate.WithInvokeFunc(func(ctx context.Context, i fn.Invoker) (interface{}, error) {
					return "nope", nil
				}),
			},
			ExpectedValue: "nope",
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
			Name: "traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{
						"bar": map[string]interface{}{
							"bar": map[string]interface{}{"baz": "quux"},
						},
					},
				)),
			},
			ExpectedValue: "quux",
		},
		{
			Name: "JSONPath traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "$.foo.bar.baz",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPath),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{
						"bar": map[string]interface{}{
							"bar": map[string]interface{}{"baz": "quux"},
						},
					},
				)),
			},
			ExpectedValue: "quux",
		},
		{
			Name: "JSONPath traverses output",
			Data: `{
				"foo": {"$type": "Output", "from": "baz", "name": "bar"}
			}`,
			Query: "$.foo.b[1]",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPath),
				evaluate.WithOutputTypeResolver(resolve.NewMemoryOutputTypeResolver(
					map[resolve.MemoryOutputKey]interface{}{
						{From: "baz", Name: "bar"}: map[string]interface{}{
							"a": "test",
							"b": []interface{}{"c", "d"},
						},
					},
				)),
			},
			ExpectedValue: "d",
		},
		{
			Name: "JSONPath template traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "{.foo.bar.baz}",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPathTemplate),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{
						"bar": map[string]interface{}{
							"bar": map[string]interface{}{"baz": "quux"},
						},
					},
				)),
			},
			ExpectedValue: "quux",
		},
		{
			Name: "unresolvable",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			ExpectedUnresolvable: evaluate.Unresolvable{
				Parameters: []evaluate.UnresolvableParameter{
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
			Query: "$.*.value",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPath),
			},
			ExpectedValue: []interface{}{"gggggg"},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"foo": "very secret"},
				)),
			},
			ExpectedValue: "very secret",
		},
		{
			Name: "JSONPath unresolvable not evaluated because not in path",
			Data: `{
				"a": {"name": "aa", "value": {"$type": "Parameter", "name": "bar"}},
				"b": {"name": "bb", "value": {"$type": "Secret", "name": "foo"}}
			}`,
			Query: "$.*.name",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPath),
			},
			ExpectedValue: randomOrder{"aa", "bb"},
		},
		{
			Name: "type resolver returns an unsupported type",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"}
			}`,
			Query: "a.inner",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
					"foo": map[string]string{"inner": "test"},
				})),
			},
			ExpectedError: &evaluate.UnsupportedValueError{
				Type: reflect.TypeOf(map[string]string(nil)),
			},
		},
		{
			Name: "type resolver returns an unsupported type in JSONPath",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"},
				"b": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "$.a.inner",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPath),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
					"foo": map[string]string{"inner": "test"},
					"bar": map[string]interface{}{"inner": "test"},
				})),
			},
			ExpectedError: &evaluate.UnsupportedValueError{
				Type: reflect.TypeOf(map[string]string(nil)),
			},
		},
		{
			Name: "type resolver returns an unsupported type in JSONPath template",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"}
			}`,
			Query: "{.a.inner}",
			Opts: []evaluate.Option{
				evaluate.WithLanguage(evaluate.LanguageJSONPathTemplate),
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(map[string]interface{}{
					"foo": map[string]string{"inner": "test"},
				})),
			},
			ExpectedError: &evaluate.UnsupportedValueError{
				Type: reflect.TypeOf(map[string]string(nil)),
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"foo": "v3ry s3kr3t!"},
				)),
			},
			Into:          &root{},
			ExpectedValue: &root{Foo: foo{Bar: "v3ry s3kr3t!"}},
		},
		{
			Name:          "unresolvable",
			Data:          `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Into:          &root{Foo: foo{Bar: "masked"}},
			ExpectedValue: &root{},
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
					{Name: "foo"},
				},
			},
		},
		{
			Name: "map",
			Data: `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"foo": "v3ry s3kr3t!"},
				)),
			},
			Into:          &map[string]interface{}{},
			ExpectedValue: &map[string]interface{}{"foo": map[string]interface{}{"bar": "v3ry s3kr3t!"}},
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
			ExpectedUnresolvable: evaluate.Unresolvable{
				Secrets: []evaluate.UnresolvableSecret{
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{
						"aws.accessKeyID":     "AKIANOAHISCOOL",
						"aws.secretAccessKey": "abcdefs3cr37s",
					},
				)),
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
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{
						"aws": `{
							"accessKeyID": "AKIANOAHISCOOL",
							"secretAccessKey": "abcdefs3cr37s",
							"region": "us-west-2",
							"extra": "unused"
						}`,
					},
				)),
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

func TestJSON(t *testing.T) {
	tests{
		{
			Name: "encoded safe string",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "SGVsbG8sIHdvcmxkIQ=="
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithResultMapper(evaluate.NewJSONResultMapper()),
			},
			ExpectedValue: []byte(`{"foo":"Hello, world!"}`),
		},
		{
			Name: "encoded unsafe string",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "SGVsbG8sIJCiikU="
				}
			}`,
			Opts: []evaluate.Option{
				evaluate.WithResultMapper(evaluate.NewJSONResultMapper()),
			},
			ExpectedValue: []byte(`{"foo":{"$encoding":"base64","data":"SGVsbG8sIJCiikU="}}`),
		},
	}.RunAll(t)
}
