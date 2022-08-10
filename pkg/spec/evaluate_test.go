package spec_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
	"github.com/puppetlabs/leg/gvalutil/pkg/template"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/fnlib"
	"github.com/puppetlabs/leg/relspec/pkg/pathlang"
	"github.com/puppetlabs/leg/relspec/pkg/query"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
	"github.com/puppetlabs/leg/relspec/pkg/relspec"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type randomOrder []any

type test struct {
	Name               string
	Data               string
	Opts               []spec.Option
	Depth              int
	QueryLanguage      query.Language[*spec.References]
	Query              string
	Into               any
	ExpectedValue      any
	ExpectedReferences *spec.References
	ExpectedError      error
}

func (tt test) Run(t *testing.T) {
	ctx := context.Background()

	tree, err := spec.ParseJSONString(tt.Data)
	require.NoError(t, err)

	ev := spec.NewEvaluator(tt.Opts...)

	check := func(t *testing.T, err error) {
		if tt.ExpectedError != nil {
			require.Equal(t, tt.ExpectedError, err)
		} else {
			require.NoError(t, err)
		}
	}

	var r *evaluate.Result[*spec.References]
	if tt.Query != "" {
		lang := tt.QueryLanguage
		if lang == nil {
			lang = pathlang.New[*spec.References](
				pathlang.WithFunctionMap[*spec.References]{Map: fnlib.Library[*spec.References]()},
			).Expression
		}

		r, err = query.EvaluateQuery(ctx, ev, lang, tree, tt.Query)
		check(t, err)
	} else if tt.Into != nil {
		md, err := evaluate.EvaluateInto(ctx, ev, tree, tt.Into)
		check(t, err)

		r = evaluate.NewResult(md, tt.Into)
	} else {
		depth := tt.Depth
		if depth == 0 {
			depth = -1
		}

		r, err = ev.Evaluate(ctx, tree, depth)
		check(t, err)
	}

	require.Equal(t, tt.ExpectedError == nil, r != nil)
	if r == nil {
		return
	}

	expected := tt.ExpectedValue
	if ro, ok := expected.(randomOrder); ok {
		expected = []any(ro)

		// Requests sorting before continuing.
		if actual, ok := r.Value.([]any); ok {
			sort.Slice(actual, func(i, j int) bool {
				return fmt.Sprintf("%T %v", actual[i], actual[i]) < fmt.Sprintf("%T %v", actual[j], actual[j])
			})
		}
	}

	assert.Equal(t, expected, r.Value)

	refs := tt.ExpectedReferences
	if refs == nil {
		refs = spec.NewReferences()
	}

	assert.Equal(t, refs, r.References)
}

type tests []test

func (tts tests) RunAll(t *testing.T) {
	for _, tt := range tts {
		t.Run(tt.Name, tt.Run)
	}
}

func TestEvaluate(t *testing.T) {
	tests{
		{
			Name: "invalid data resolver",
			Data: `{"baz": {"$type": "Data", "query": "foo.bar"}}`,
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "baz",
				Cause: &relspec.InvalidTypeError{
					Type:  "Data",
					Cause: &spec.DataResolverNotFoundError{},
				},
			},
		},
		{
			Name: "invalid data resolver in template",
			Data: `{"baz": "${event.foo.bar}"}`,
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "baz",
				Cause: &template.EvaluationError{
					Start: "${",
					Cause: &evaluate.PathEvaluationError{
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
			ExpectedValue: map[string]any{
				"foo": jsonSecret("bar"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable secret in template",
			Data: `{"foo": "${secrets.bar}"}`,
			ExpectedValue: map[string]any{
				"foo": "${secrets.bar}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable connection",
			Data: `{"foo": {"$type": "Connection", "type": "blort", "name": "bar"}}`,
			ExpectedValue: map[string]any{
				"foo": jsonConnection("blort", "bar"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable connection in template",
			Data: `{"foo": "${connections.blort.bar}"}`,
			ExpectedValue: map[string]any{
				"foo": "${connections.blort.bar}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable output",
			Data: `{"foo": {"$type": "Output", "from": "baz", "name": "bar"}}`,
			ExpectedValue: map[string]any{
				"foo": jsonOutput("baz", "bar"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.Errored(spec.OutputID{From: "baz", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable output in template",
			Data: `{"foo": "${outputs.baz.bar}"}`,
			ExpectedValue: map[string]any{
				"foo": "${outputs.baz.bar}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.Errored(spec.OutputID{From: "baz", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable parameter",
			Data: `{"foo": {"$type": "Parameter", "name": "bar"}}`,
			ExpectedValue: map[string]any{
				"foo": jsonParameter("bar"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable parameter in template",
			Data: `{"foo": "${parameters.bar}"}`,
			ExpectedValue: map[string]any{
				"foo": "${parameters.bar}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "invalid data",
			Data: `{"foo": [{"$type": "Data"}]}`,
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "foo",
				Cause: &evaluate.PathEvaluationError{
					Path: "0",
					Cause: &relspec.InvalidTypeError{
						Type:  "Data",
						Cause: &spec.FieldNotFoundError{Name: "query"},
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
					Cause: &relspec.InvalidTypeError{
						Type:  "Secret",
						Cause: &spec.FieldNotFoundError{Name: "name"},
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
					Cause: &relspec.InvalidTypeError{
						Type:  "Connection",
						Cause: &spec.FieldNotFoundError{Name: "type"},
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
					Cause: &relspec.InvalidTypeError{
						Type:  "Output",
						Cause: &spec.FieldNotFoundError{Name: "from"},
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
					Cause: &relspec.InvalidTypeError{
						Type:  "Parameter",
						Cause: &spec.FieldNotFoundError{Name: "name"},
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
					Cause: &relspec.InvalidEncodingError{
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
			Opts: []spec.Option{
				spec.WithDataTypeResolver{
					DataTypeResolver: spec.NewMemoryDataTypeResolver(
						map[string]any{"foo": map[string]string{"bar": "baz"}},
					),
				},
			},
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "data",
				Cause: &relspec.InvalidTypeError{
					Type: "Data",
					Cause: &spec.DataQueryError{
						Query: "fo{o.b}ar",
						Cause: fmt.Errorf("parsing error: fo{o.b}ar\t:1:3 - 1:4 unexpected \"{\" while scanning operator"),
					},
				},
			},
		},
		{
			Name: "many unresolvable",
			Data: `{
				"a": {"$type": "Secret", "name": "foo"},
				"b": {"$type": "Output", "from": "baz", "name": "bar"},
				"c": {"$type": "Parameter", "name": "quux"},
				"d": "hello",
				"e": {"$type": "Answer", "askRef": "baz", "name": "bar"},
				"f": {"$type": "Connection", "type": "blort", "name": "bar"}
			}`,
			ExpectedValue: map[string]any{
				"a": jsonSecret("foo"),
				"b": jsonOutput("baz", "bar"),
				"c": jsonParameter("quux"),
				"d": "hello",
				"e": jsonAnswer("baz", "bar"),
				"f": jsonConnection("blort", "bar"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "foo"}, spec.ErrNotFound))
				r.Outputs.Set(ref.Errored(spec.OutputID{From: "baz", Name: "bar"}, spec.ErrNotFound))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
				r.Answers.Set(ref.Errored(spec.AnswerID{AskRef: "baz", Name: "bar"}, spec.ErrNotFound))
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "many unresolvable in template",
			Data: `{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "hello",
				"e": "${connections.blort.bar}"
			}`,
			ExpectedValue: map[string]any{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "hello",
				"e": "${connections.blort.bar}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "foo"}, spec.ErrNotFound))
				r.Outputs.Set(ref.Errored(spec.OutputID{From: "baz", Name: "bar"}, spec.ErrNotFound))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
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
			ExpectedValue: map[string]any{
				"foo": []any{
					map[string]any{"a": jsonSecret("foo")},
					jsonParameter("bar"),
				},
				"bar": jsonParameter("frob"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "bar"}, spec.ErrNotFound))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "frob"}, spec.ErrNotFound))
			}),
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
			ExpectedValue: map[string]any{
				"foo": []any{
					map[string]any{"a": "${secrets.foo}"},
					"${parameters.bar}",
				},
				"bar": "${parameters.frob}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "bar"}, spec.ErrNotFound))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "frob"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable",
			Data: `{
				"a": {"$type": "Secret", "name": "foo"},
				"b": {"$type": "Output", "from": "baz", "name": "bar"},
				"c": {"$type": "Parameter", "name": "quux"},
				"d": {"$fn.concat": ["foo", "bar"]},
				"e": "hello",
				"f": {"$type": "Answer", "askRef": "baz", "name": "bar"},
				"g": {"$type": "Data", "query": "foo.bar"},
				"h": {"$type": "Connection", "type": "blort", "name": "bar"}
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": []any{1, 2, 3}},
					),
				},
				spec.WithAnswerTypeResolver{
					AnswerTypeResolver: spec.NewMemoryAnswerTypeResolver(
						map[spec.MemoryAnswerKey]any{
							{AskRef: "baz", Name: "bar"}: "approved",
						},
					),
				},
				spec.WithDataTypeResolver{
					Name:    "event",
					Default: true,
					DataTypeResolver: spec.NewMemoryDataTypeResolver(
						map[string]any{"foo": map[string]any{"bar": "baz"}},
					),
				},
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{"bar": "blort"},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"a": "v3ry s3kr3t!",
				"b": "127.0.0.1",
				"c": []any{1, 2, 3},
				"d": "foobar",
				"e": "hello",
				"f": "approved",
				"g": "baz",
				"h": map[string]any{"bar": "blort"},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
				r.Answers.Set(ref.OK(spec.AnswerID{AskRef: "baz", Name: "bar"}))
				r.Data.Set(ref.OK(spec.DataID{}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
			}),
		},
		{
			Name: "resolvable in template",
			Data: `{
				"a": "${secrets.foo}",
				"b": "${outputs.baz.bar}",
				"c": "${parameters.quux}",
				"d": "${concat('foo', 'bar')}",
				"e": "hello",
				"g": "${event.foo.bar}",
				"h": "${connections.blort.bar}"
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": []any{1, 2, 3}},
					),
				},
				spec.WithDataTypeResolver{
					Name: "event",
					DataTypeResolver: spec.NewMemoryDataTypeResolver(
						map[string]any{"foo": map[string]any{"bar": "baz"}},
					),
				},
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{"bar": "blort"},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"a": "v3ry s3kr3t!",
				"b": "127.0.0.1",
				"c": []any{1, 2, 3},
				"d": "foobar",
				"e": "hello",
				"g": "baz",
				"h": map[string]any{"bar": "blort"},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
				r.Data.Set(ref.OK(spec.DataID{Name: "event"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
			}),
		},
		{
			Name: "resolvable expansion in template",
			Data: `{
				"foo": "${$}"
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(map[string]string{"foo": "v3ry s3kr3t!"}),
				},
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": []any{1, 2, 3}},
					),
				},
				spec.WithDataTypeResolver{
					Name: "event",
					DataTypeResolver: spec.NewMemoryDataTypeResolver(
						map[string]any{"foo": map[string]any{"bar": "baz"}},
					),
				},
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}:  map[string]any{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]any{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]any{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"secrets": map[string]any{
						"foo": "v3ry s3kr3t!",
					},
					"outputs": map[string]any{
						"baz": map[string]any{
							"bar": "127.0.0.1",
						},
					},
					"parameters": map[string]any{
						"quux": []any{1, 2, 3},
					},
					"event": map[string]any{
						"foo": map[string]any{
							"bar": "baz",
						},
					},
					"connections": map[string]any{
						"blort": map[string]any{
							"bar":  map[string]any{"bar": "blort"},
							"wish": map[string]any{"sim": "jax"},
						},
						"zup": map[string]any{
							"bar": map[string]any{"waz": "mux"},
						},
					},
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
				r.Data.Set(ref.OK(spec.DataID{Name: "event"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "wish"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "zup", Name: "bar"}))
			}),
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
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"accessKeyID": "AKIANOAHISCOOL", "secretAccessKey": "abcdefs3cr37s"},
					),
				},
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"instanceID": "i-abcdef123456"},
					),
				},
			},
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "AKIANOAHISCOOL",
					"secretAccessKey": "abcdefs3cr37s",
				},
				"instanceID": "i-abcdef123456",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "accessKeyID"}))
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "secretAccessKey"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "instanceID"}))
			}),
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
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"accessKeyID": "AKIANOAHISCOOL", "secretAccessKey": "abcdefs3cr37s"},
					),
				},
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"instanceID": "i-abcdef123456"},
					),
				},
			},
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "AKIANOAHISCOOL",
					"secretAccessKey": "abcdefs3cr37s",
				},
				"instanceID": "i-abcdef123456",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "accessKeyID"}))
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "secretAccessKey"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "instanceID"}))
			}),
		},
		{
			Name: "resolvable parameter traversal",
			Data: `{
				"accessKeyID": "${parameters.aws.accessKeyID}"
			}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"aws": map[string]any{"accessKeyID": "foo", "secretAccessKey": "bar"}},
					),
				},
			},
			ExpectedValue: map[string]any{
				"accessKeyID": "foo",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "aws"}))
			}),
		},
		{
			Name: "resolvable output traversal",
			Data: `{
				"test": "${outputs.baz.bar.b[1]}"
			}`,
			Opts: []spec.Option{
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: map[string]any{
								"a": "test",
								"b": []any{"c", "d"},
							},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"test": "d",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
			}),
		},
		{
			Name: "resolvable parameter in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "aws"}))
			}),
		},
		{
			Name: "resolvable parameter in invocation argument in partial template",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": "${parameters.aws}"}
			}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "aws"}))
			}),
		},
		{
			Name: "resolvable parameter in invocation argument in template",
			Data: `{
				"aws": "${jsonUnmarshal(parameters.aws)}"
			}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"aws": `{"accessKeyID": "foo", "secretAccessKey": "bar"}`},
					),
				},
			},
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID":     "foo",
					"secretAccessKey": "bar",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "aws"}))
			}),
		},
		{
			Name: "unresolvable parameter in invocation argument",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Parameter", "name": "aws"}}
			}`,
			ExpectedValue: map[string]any{
				"aws": jsonInvocation("jsonUnmarshal", []any{jsonParameter("aws")}),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "aws"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable parameter in invocation argument in partial template",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": "${parameters.aws}"}
			}`,
			ExpectedValue: map[string]any{
				"aws": jsonInvocation("jsonUnmarshal", []any{"${parameters.aws}"}),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "aws"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable parameter in invocation argument in template",
			Data: `{
				"aws": "${jsonUnmarshal(parameters.aws)}"
			}`,
			ExpectedValue: map[string]any{
				"aws": "${jsonUnmarshal(parameters.aws)}",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "aws"}, spec.ErrNotFound))
			}),
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
			ExpectedValue: map[string]any{
				"foo": jsonInvocation("concat", []any{
					"bar",
					jsonParameter("second"),
				}),
			},
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"first": "bar"},
					),
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "first"}))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "second"}, spec.ErrNotFound))
			}),
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
			ExpectedValue: map[string]any{
				"foo": jsonInvocation("concat", []any{
					"bar",
					"${parameters.second}",
				}),
			},
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"first": "bar"},
					),
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "first"}))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "second"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "partially resolvable invocation in template",
			Data: `{
				"foo": "${concat(parameters.first, parameters.second)}"
			}`,
			ExpectedValue: map[string]any{
				"foo": "${concat(parameters.first, parameters.second)}",
			},
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"first": "bar"},
					),
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "first"}))
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "second"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolved conditionals evaluation",
			Data: `{
				"conditions": [{"$fn.equals": [
					{"$type": "Parameter", "name": "first"},
					"foobar"
				]}]
			}`,
			ExpectedValue: map[string]any{
				"conditions": []any{jsonInvocation("equals", []any{
					jsonParameter("first"),
					"foobar",
				})},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "first"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolved conditionals evaluation in template",
			Data: `{
				"conditions": ["${parameters.first == 'foobar'}"]
			}`,
			ExpectedValue: map[string]any{
				"conditions": []any{
					"${parameters.first == 'foobar'}",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "first"}, spec.ErrNotFound))
			}),
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
			ExpectedValue: map[string]any{
				"conditions": []any{true, true},
			},
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"first": "foobar"},
					),
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "first"}))
			}),
		},
		{
			Name: "resolved conditionals evaluation in template",
			Data: `{
				"conditions": [
					"${parameters.first == 'foobar'}",
					"${parameters.first != 'barfoo'}"
				]
			}`,
			ExpectedValue: map[string]any{
				"conditions": []any{true, true},
			},
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"first": "foobar"},
					),
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "first"}))
			}),
		},
		{
			Name: "complex conditionals",
			Data: `"${parameters.comp == outputs.baz && outputs.baz.id < 'second'}"`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"comp": map[string]any{"ip": "127.0.0.1", "id": "first"},
						},
					),
				},
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "ip"}: "127.0.0.1",
							{From: "baz", Name: "id"}: "first",
						},
					),
				},
			},
			ExpectedValue: true,
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "comp"}))
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "id"}))
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "ip"}))
			}),
		},
		{
			Name: "encoded string from secret",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": {"$type": "Secret", "name": "bar"}
				}
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"bar": "SGVsbG8sIJCiikU="},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": "Hello, \x90\xA2\x8A\x45",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "bar"}))
			}),
		},
		{
			Name: "encoded string from secret in template",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "${secrets.bar}"
				}
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"bar": "SGVsbG8sIJCiikU="},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": "Hello, \x90\xA2\x8A\x45",
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "bar"}))
			}),
		},
		{
			Name: "encoded string from unresolvable secret",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": {"$type": "Secret", "name": "bar"}
				}
			}`,
			ExpectedValue: map[string]any{
				"foo": jsonEncoding(transfer.Base64EncodingType, jsonSecret("bar")),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "encoded string from unresolvable secret in template",
			Data: `{
				"foo": {
					"$encoding": "base64",
					"data": "${secrets.bar}"
				}
			}`,
			ExpectedValue: map[string]any{
				"foo": jsonEncoding(transfer.Base64EncodingType, "${secrets.bar}"),
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable template dereferencing",
			Data: `"${secrets['regions.' + parameters.region]}"`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"region": "us-east-1",
						},
					),
				},
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: "EAST",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "region"}))
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "regions.us-east-1"}))
			}),
		},
		{
			Name: "unresolvable template dereferencing",
			Data: `"${secrets['regions.' + parameters.region]}"`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"region": "us-east-2",
						},
					),
				},
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: `${secrets['regions.' + parameters.region]}`,
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "region"}))
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "regions.us-east-2"}, spec.ErrNotFound))
			}),
		},
		{
			Name:          "nested unresolvable template dereferencing",
			Data:          `"${secrets['regions.' + parameters.region]}"`,
			ExpectedValue: `${secrets['regions.' + parameters.region]}`,
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "region"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "parameters expansion in template",
			Data: `{
				"foo": "${parameters}"
			}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"region": "us-east-1",
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"region": "us-east-1",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "region"}))
			}),
		},
		{
			Name: "secrets expansion in template",
			Data: `{
				"foo": "${secrets}"
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{
							"regions.us-east-1": "EAST",
							"regions.us-west-1": "WEST",
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"regions.us-east-1": "EAST",
					"regions.us-west-1": "WEST",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "regions.us-east-1"}))
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "regions.us-west-1"}))
			}),
		},
		{
			Name: "outputs expansion in template",
			Data: `{
				"foo": "${outputs}"
			}`,
			Opts: []spec.Option{
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"baz": map[string]any{
						"bar": "127.0.0.1",
					},
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
			}),
		},
		{
			Name: "single step output expansion in template",
			Data: `{
				"foo": "${outputs.baz}"
			}`,
			Opts: []spec.Option{
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: "127.0.0.1",
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"bar": "127.0.0.1",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
			}),
		},
		{
			Name: "connections expansion in template",
			Data: `{
				"foo": "${connections}"
			}`,
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}:  map[string]any{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]any{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]any{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"blort": map[string]any{
						"bar":  map[string]any{"bar": "blort"},
						"wish": map[string]any{"sim": "jax"},
					},
					"zup": map[string]any{
						"bar": map[string]any{"waz": "mux"},
					},
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "zup", Name: "bar"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "wish"}))
			}),
		},
		{
			Name: "single connection type expansion in template",
			Data: `{
				"foo": "${connections.blort}"
			}`,
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}:  map[string]any{"bar": "blort"},
							{Type: "zup", Name: "bar"}:    map[string]any{"waz": "mux"},
							{Type: "blort", Name: "wish"}: map[string]any{"sim": "jax"},
						},
					),
				},
			},
			ExpectedValue: map[string]any{
				"foo": map[string]any{
					"bar":  map[string]any{"bar": "blort"},
					"wish": map[string]any{"sim": "jax"},
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "wish"}))
			}),
		},
	}.RunAll(t)
}

func TestEvaluateQuery(t *testing.T) {
	tests{
		{
			Name: "traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"bar": map[string]any{
								"bar": map[string]any{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "bar"}))
			}),
		},
		{
			Name: "JSONPath traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.foo.bar.baz",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"bar": map[string]any{
								"bar": map[string]any{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "bar"}))
			}),
		},
		{
			Name: "JSONPath traverses output",
			Data: `{
				"foo": {"$type": "Output", "from": "baz", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.foo.b[1]",
			Opts: []spec.Option{
				spec.WithOutputTypeResolver{
					OutputTypeResolver: spec.NewMemoryOutputTypeResolver(
						map[spec.MemoryOutputKey]any{
							{From: "baz", Name: "bar"}: map[string]any{
								"a": "test",
								"b": []any{"c", "d"},
							},
						},
					),
				},
			},
			ExpectedValue: "d",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Outputs.Set(ref.OK(spec.OutputID{From: "baz", Name: "bar"}))
			}),
		},
		{
			Name: "JSONPath template traverses parameter",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage[*spec.References],
			Query:         "{.foo.bar.baz}",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{
							"bar": map[string]any{
								"bar": map[string]any{"baz": "quux"},
							},
						},
					),
				},
			},
			ExpectedValue: "quux",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "bar"}))
			}),
		},
		{
			Name: "unresolvable",
			Data: `{
				"foo": {"$type": "Parameter", "name": "bar"}
			}`,
			Query: "foo.bar.baz",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "JSONPath unresolvable",
			Data: `{
				"a": {"name": "aa", "value": {"$type": "Secret", "name": "foo"}},
				"b": {"name": "bb", "value": {"$type": "Secret", "name": "bar"}},
				"c": {"name": "cc", "value": "gggggg"}
			}`,
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.*.value",
			ExpectedValue: []any{"gggggg"},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "foo"}, spec.ErrNotFound))
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable not evaluated because not in path",
			Data: `{
				"a": {"$type": "Parameter", "name": "bar"},
				"b": {"c": {"$type": "Secret", "name": "foo"}}
			}`,
			Query: "b.c",
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"foo": "very secret"},
					),
				},
			},
			ExpectedValue: "very secret",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
			}),
		},
		{
			Name: "JSONPath object unresolvable not evaluated because not in path",
			Data: `{
				"a": {"name": "aa", "value": {"$type": "Parameter", "name": "bar"}},
				"b": {"name": "bb", "value": {"$type": "Secret", "name": "foo"}}
			}`,
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.*.name",
			ExpectedValue: randomOrder{"aa", "bb"},
		},
		{
			Name: "JSONPath array unresolvable not evaluated because not in path",
			Data: `[
				{"name": "aa", "value": {"$type": "Parameter", "name": "bar"}},
				{"name": "bb", "value": {"$type": "Secret", "name": "foo"}}
			]`,
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.*.name",
			ExpectedValue: randomOrder{"aa", "bb"},
		},
		{
			Name: "type resolver returns an unsupported type",
			Data: `{
				"a": {"$type": "Parameter", "name": "foo"}
			}`,
			Query: "a.inner",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(map[string]any{
						"foo": map[string]string{"inner": "test"},
					}),
				},
			},
			ExpectedError: &evaluate.PathEvaluationError{
				Path: "a",
				Cause: &evaluate.PathEvaluationError{
					Path: "inner",
					Cause: &evaluate.UnsupportedValueError{
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
			QueryLanguage: query.JSONPathLanguage[*spec.References],
			Query:         "$.a.inner",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(map[string]any{
						"foo": map[string]string{"inner": "test"},
						"bar": map[string]any{"inner": "test"},
					}),
				},
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
			QueryLanguage: query.JSONPathTemplateLanguage[*spec.References],
			Query:         "{.a.inner}",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(map[string]any{
						"foo": map[string]string{"inner": "test"},
					}),
				},
			},
			ExpectedError: &template.EvaluationError{
				Start: "{",
				Cause: &evaluate.UnsupportedValueError{
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
			QueryLanguage: query.JSONPathTemplateLanguage[*spec.References],
			Query:         "{.args}",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(map[string]any{
						"deployment": "my-test-deployment",
					}),
				},
			},
			ExpectedValue: "map[a:undo b:deployment.v1.apps/my-test-deployment]",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "deployment"}))
			}),
		},
		{
			Name: "JSONPath template traverses array",
			Data: `{
				"args": [
					"undo",
					{"$fn.concat": ["deployment.v1.apps/", {"$type": "Parameter", "name": "deployment"}]}
				]
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage[*spec.References],
			Query:         "{.args}",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(map[string]any{
						"deployment": "my-test-deployment",
					}),
				},
			},
			ExpectedValue: "undo deployment.v1.apps/my-test-deployment",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "deployment"}))
			}),
		},
		{
			Name: "JSONPath template traverses array with unresolvables",
			Data: `{
				"args": [
					"undo",
					{"$fn.concat": ["deployment.v1.apps/", {"$type": "Parameter", "name": "deployment"}]}
				]
			}`,
			QueryLanguage: query.JSONPathTemplateLanguage[*spec.References],
			Query:         "{.args}",
			ExpectedValue: "undo map[$fn.concat:[deployment.v1.apps/ map[$type:Parameter name:deployment]]]",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "deployment"}, spec.ErrNotFound))
			}),
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
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"foo": "v3ry s3kr3t!"},
					),
				},
			},
			Into:          &root{},
			ExpectedValue: &root{Foo: foo{Bar: "v3ry s3kr3t!"}},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
			}),
		},
		{
			Name:          "unresolvable",
			Data:          `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Into:          &root{Foo: foo{Bar: "masked"}},
			ExpectedValue: &root{},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "foo"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "map",
			Data: `{"foo": {"bar": {"$type": "Secret", "name": "foo"}}}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"foo": "v3ry s3kr3t!"},
					),
				},
			},
			Into:          &map[string]any{},
			ExpectedValue: &map[string]any{"foo": map[string]any{"bar": "v3ry s3kr3t!"}},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "foo"}))
			}),
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
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "aws.accessKeyID"}, spec.ErrNotFound))
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "aws.secretAccessKey"}, spec.ErrNotFound))
			}),
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
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
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
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "aws.accessKeyID"}))
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "aws.secretAccessKey"}))
			}),
		},
		{
			Name: "resolvable traverses",
			Data: `{
				"aws": {"$fn.jsonUnmarshal": {"$type": "Secret", "name": "aws"}},
				"op": {"$type": "Parameter", "name": "op"}
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
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
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "aws"}))
			}),
		},
	}.RunAll(t)
}

func TestPath(t *testing.T) {
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
			Name: "resolvable (using connections) into struct",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$type": "Connection", "name": "aws", "type": "aws"}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "aws", Name: "aws"}: map[string]any{
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
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "aws", Name: "aws"}))
			}),
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
			ExpectedValue: map[string]any(
				map[string]any{
					"aws": map[string]any{
						"accessKeyID": map[string]any{
							"$fn.path": map[string]any{
								"object": map[string]any{
									"$type": "Connection", "name": "aws", "type": "aws"},
								"query": "accessKeyID"}},
						"secretAccessKey": map[string]any{
							"$fn.path": map[string]any{
								"object": map[string]any{
									"$type": "Connection", "name": "aws", "type": "aws"},
								"query": "secretAccessKey"}},
						"region": "us-west-2",
					},
				},
			),
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "aws", Name: "aws"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable (using secrets) into struct",
			Data: `{
				"aws": {
					"accessKeyID": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "accessKeyID"}},
					"secretAccessKey": {"$fn.path": {"object": {"$fn.jsonUnmarshal": [{"$type": "Secret", "name": "aws"}]}, "query": "secretAccessKey"}},
					"region": "us-west-2"
				}
			}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
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
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{"aws"}))
			}),
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
			ExpectedValue: map[string]any{
				"aws": map[string]any{
					"accessKeyID": map[string]any{
						"$fn.path": map[string]any{
							"object": map[string]any{
								"$fn.jsonUnmarshal": []any{map[string]any{
									"$type": "Secret", "name": "aws"}}},
							"query": "accessKeyID"}},
					"secretAccessKey": map[string]any{
						"$fn.path": map[string]any{
							"object": map[string]any{
								"$fn.jsonUnmarshal": []any{map[string]any{
									"$type": "Secret", "name": "aws"}}},
							"query": "secretAccessKey"}},
					"region": "us-west-2",
				},
			},
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "aws"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable (using secrets) with expression language and path exists",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${jsonUnmarshal(secrets.blort)}"
					}
				},
				"foo.bar.grault.garply"
			]}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "blort"}))
			}),
		},
		{
			Name: "resolvable (using secrets) and path does not exist",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${jsonUnmarshal(secrets.blort)}"
					}
				},
				"foo.baz.grault.garply",
				"xyzzy"
			]}`,
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
		},
		{
			Name: "unresolvable (using secrets) with expression language",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${jsonUnmarshal(secrets.blort)}"
					}
				},
				"foo.bar.grault"
			]}`,
			ExpectedValue: jsonInvocation("path", []any{
				map[string]any{"foo": map[string]any{"bar": "${jsonUnmarshal(secrets.blort)}"}},
				"foo.bar.grault",
			}),
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "blort"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable (using connections) and path exists",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${connections.blort.bar}"
					}
				},
				"foo.bar.grault"
			]}`,
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{
								"quuz":   "quux",
								"grault": "garply",
							},
						},
					),
				},
			},
			ExpectedValue: "garply",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
			}),
		},
		{
			Name: "resolvable (using connections) and path does not exist",
			Data: `{"$fn.path": [
				{
					"foo": {
						"baz": "${connections.blort.bar}"
					}
				},
				"foo.bar.grault",
				"xyzzy"
			]}`,
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{
								"quuz":   "quux",
								"grault": "garply",
							},
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
		},
		{
			Name: "unresolvable (using connections)",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${connections.blort.bar}"
					}
				},
				"foo.bar.grault",
				"xyzzy"
			]}`,
			ExpectedValue: jsonInvocation("path", []any{
				map[string]any{"foo": map[string]any{"bar": "${connections.blort.bar}"}},
				"foo.bar.grault",
				"xyzzy",
			}),
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable (using parameters) and path exists",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${parameters.quux}"
					}
				},
				"foo.bar"
			]}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz"},
					),
				},
			},
			ExpectedValue: "baz",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
			}),
		},
		{
			Name: "path and query are resolvable (using parameters) and path exists",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${parameters.quux}"
					}
				},
				"${parameters.quuz}"
			]}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz", "quuz": "foo.bar"},
					),
				},
			},
			ExpectedValue: "baz",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quuz"}))
			}),
		},
		{
			Name: "resolvable (using parameters) and path does not exist",
			Data: `{"$fn.path": [
				{
					"foo": {
						"quuz": "${parameters.quux}"
					}
				},
				"foo.bar",
				"grault"
			]}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz"},
					),
				},
			},
			ExpectedValue: "grault",
		},
		{
			Name: "path and default are resolvable (using parameters) and path does not exist",
			Data: `{"$fn.path": [
				{
					"foo": {
						"quuz": "${parameters.quux}"
					}
				},
				"foo.bar",
				"${parameters.grault}"
			]}`,
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz", "grault": "garply"},
					),
				},
			},
			ExpectedValue: "garply",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "grault"}))
			}),
		},
		{
			Name: "unresolvable (using parameters)",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${parameters.quux}"
					}
				},
				"foo.bar"
			]}`,
			ExpectedValue: jsonInvocation("path", []any{
				map[string]any{"foo": map[string]any{"bar": "${parameters.quux}"}},
				"foo.bar",
			}),
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "unresolvable (using parameters) with default",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${parameters.quux}"
					}
				},
				"foo.bar",
				42
			]}`,
			ExpectedValue: jsonInvocation("path", []any{
				map[string]any{"foo": map[string]any{"bar": "${parameters.quux}"}},
				"foo.bar",
				42.,
			}),
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "resolvable (using parameters) but object is not completely resolvable",
			Data: `{"$fn.path": [
				{
					"foo": {
						"bar": "${parameters.quux}",
						"baz": "ok"
					}
				},
				"foo.baz",
				42
			]}`,
			ExpectedValue: "ok",
		},
	}.RunAll(t)
}

func TestPathWithExpressions(t *testing.T) {
	tests{
		{
			Name: "path is resolvable (using secrets) and path exists",
			Data: `{
				"foo": {
					"bar": "${jsonUnmarshal(secrets.blort)}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar.grault.garply')",
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.OK(spec.SecretID{Name: "blort"}))
			}),
		},
		{
			Name: "path is resolvable (using secrets) and path does not exist",
			Data: `{
				"foo": {
					"bar": "${jsonUnmarshal(secrets.blort)}"
				}
			}`,
			Query: "path(object: $, query: 'foo.baz.grault.garply', default: 'xyzzy')",
			Opts: []spec.Option{
				spec.WithSecretTypeResolver{
					SecretTypeResolver: spec.NewMemorySecretTypeResolver(
						map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
		},
		{
			Name: "path is not resolvable (using secrets)",
			Data: `{
				"foo": {
					"bar": "${jsonUnmarshal(secrets.blort)}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar.grault')",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Secrets.Set(ref.Errored(spec.SecretID{Name: "blort"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "path is resolvable (using connections) and path exists",
			Data: `{
				"foo": {
					"bar": "${connections.blort.bar}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar.grault')",
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{
								"quuz":   "quux",
								"grault": "garply",
							},
						},
					),
				},
			},
			ExpectedValue: "garply",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.OK(spec.ConnectionID{Type: "blort", Name: "bar"}))
			}),
		},
		{
			Name: "path is resolvable (using connections) and path does not exist",
			Data: `{
				"foo": {
					"baz": "${connections.blort.bar}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar.grault', default: 'xyzzy')",
			Opts: []spec.Option{
				spec.WithConnectionTypeResolver{
					ConnectionTypeResolver: spec.NewMemoryConnectionTypeResolver(
						map[spec.MemoryConnectionKey]any{
							{Type: "blort", Name: "bar"}: map[string]any{
								"quuz":   "quux",
								"grault": "garply",
							},
						},
					),
				},
			},
			ExpectedValue: "xyzzy",
		},
		{
			Name: "path is not resolvable (using connections)",
			Data: `{
				"foo": {
					"bar": "${connections.blort.bar}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar.grault', default: 'xyzzy')",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Connections.Set(ref.Errored(spec.ConnectionID{Type: "blort", Name: "bar"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "path is resolvable (using parameters) and path exists",
			Data: `{
				"foo": {
					"bar": "${parameters.quux}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar')",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz"},
					),
				},
			},
			ExpectedValue: "baz",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.OK(spec.ParameterID{Name: "quux"}))
			}),
		},
		{
			Name: "path is resolvable (using parameters) and path does not exist",
			Data: `{
				"foo": {
					"quuz": "${parameters.quux}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar', default: 'grault')",
			Opts: []spec.Option{
				spec.WithParameterTypeResolver{
					ParameterTypeResolver: spec.NewMemoryParameterTypeResolver(
						map[string]any{"quux": "baz"},
					),
				},
			},
			ExpectedValue: "grault",
		},
		{
			Name: "path is not resolvable (using parameters)",
			Data: `{
				"foo": {
					"bar": "${parameters.quux}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar')",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "path is not resolvable (using parameters) with default",
			Data: `{
				"foo": {
					"bar": "${parameters.quux}"
				}
			}`,
			Query: "path(object: $, query: 'foo.bar', default: 42)",
			ExpectedReferences: spec.InitialReferences(func(r *spec.References) {
				r.Parameters.Set(ref.Errored(spec.ParameterID{Name: "quux"}, spec.ErrNotFound))
			}),
		},
		{
			Name: "path is resolvable (using parameters) but object is not completely resolvable",
			Data: `{
				"foo": {
					"bar": "${parameters.quux}",
					"baz": "ok"
				}
			}`,
			Query:         "path(object: $, query: 'foo.baz', default: 42)",
			ExpectedValue: "ok",
		},
	}.RunAll(t)
}
