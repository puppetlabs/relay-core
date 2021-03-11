package fnlib_test

import (
	"context"
	"math/cmplx"
	"reflect"
	"testing"

	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/stretchr/testify/require"
)

type test struct {
	Name                    string
	ObjectArg               interface{}
	QArg                    interface{}
	DefaultArg              interface{}
	Opts                    []evaluate.Option
	Expected                interface{}
	ExpectedIncomplete      bool
	ExpectedPositionalError error
	ExpectedKeywordError    error
}

type tests []test

func (tts tests) RunAll(t *testing.T) {
	for _, tt := range tts {
		t.Run(tt.Name, tt.Run)
	}
}

func TestPath(t *testing.T) {
	tests{
		{
			Name: "path exists",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:     "foo.bar",
			Expected: "baz",
		},
		{
			Name: "path does not exist",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg: "foo.quux",
			ExpectedPositionalError: &fn.PositionalArgError{
				Arg: 1,
				Cause: &evaluate.PathEvaluationError{
					Path: "foo",
					Cause: &evaluate.PathEvaluationError{
						Path:  "quux",
						Cause: &jsonpath.UnknownKeyError{Key: "quux"},
					},
				},
			},
			ExpectedKeywordError: &fn.KeywordArgError{
				Arg: "object",
				Cause: &evaluate.PathEvaluationError{
					Path: "foo",
					Cause: &evaluate.PathEvaluationError{
						Path:  "quux",
						Cause: &jsonpath.UnknownKeyError{Key: "quux"},
					},
				},
			},
		},
		{
			Name: "path exists with default",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:       "foo.bar",
			DefaultArg: "quux",
			Expected:   "baz",
		},
		{
			Name: "path does not exist with default",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:       "foo.quux",
			DefaultArg: 42,
			Expected:   42,
		},
		{
			Name: "path is resolvable (using secrets) and path exists",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONInvocation("jsonUnmarshal",
						[]interface{}{testutil.JSONSecret("blort")},
					),
				},
			},
			QArg: "foo.bar.grault.garply",
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
					},
				)),
			},
			Expected: "xyzzy",
		},
		{
			Name: "path is resolvable (using secrets) and path does not exist",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONInvocation("jsonUnmarshal",
						[]interface{}{testutil.JSONSecret("blort")},
					),
				},
			},
			QArg: "foo.baz.grault.garply",
			Opts: []evaluate.Option{
				evaluate.WithSecretTypeResolver(resolve.NewMemorySecretTypeResolver(
					map[string]string{"blort": `{
						"grault": {
							"garply": "xyzzy"
						}
					}`,
					},
				)),
			},
			DefaultArg: "xyzzy",
			Expected:   "xyzzy",
		},
		{
			Name: "path is not resolvable (using secrets)",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONInvocation("jsonUnmarshal",
						[]interface{}{testutil.JSONSecret("blort")},
					),
				},
			},
			QArg:               "foo.bar.grault",
			ExpectedIncomplete: true,
		},
		{
			Name: "path is resolvable (using connections) and path exists",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONConnection("blort", "bar"),
				},
			},
			QArg: "foo.bar.grault",
			Opts: []evaluate.Option{
				evaluate.WithConnectionTypeResolver(resolve.NewMemoryConnectionTypeResolver(
					map[resolve.MemoryConnectionKey]interface{}{
						{Type: "blort", Name: "bar"}: map[string]interface{}{
							"quuz":   "quux",
							"grault": "garply",
						},
					},
				)),
			},
			Expected: "garply",
		},
		{
			Name: "path is resolvable (using connections) and path does not exist",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"baz": testutil.JSONConnection("blort", "bar"),
				},
			},
			QArg: "foo.bar.grault",
			Opts: []evaluate.Option{
				evaluate.WithConnectionTypeResolver(resolve.NewMemoryConnectionTypeResolver(
					map[resolve.MemoryConnectionKey]interface{}{
						{Type: "blort", Name: "bar"}: map[string]interface{}{
							"quuz":   "quux",
							"grault": "garply",
						},
					},
				)),
			},
			DefaultArg: "xyzzy",
			Expected:   "xyzzy",
		},
		{
			Name: "path is not resolvable (using connections)",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONConnection("blort", "bar"),
				},
			},
			QArg:               "foo.bar.grault",
			DefaultArg:         "xyzzy",
			ExpectedIncomplete: true,
		},
		{
			Name: "path is resolvable (using parameters) and path exists",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg: "foo.bar",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"quux": "baz"},
				)),
			},
			Expected: "baz",
		},
		{
			Name: "path and query are resolvable (using parameters) and path exists",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg: testutil.JSONParameter("quuz"),
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"quux": "baz", "quuz": "foo.bar"},
				)),
			},
			Expected: "baz",
		},
		{
			Name: "path is resolvable (using parameters) and path does not exist",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"quuz": testutil.JSONParameter("quux"),
				},
			},
			QArg: "foo.bar",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"quux": "baz"},
				)),
			},
			DefaultArg: "grault",
			Expected:   "grault",
		},
		{
			Name: "path and default are resolvable (using parameters) and path does not exist",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"quuz": testutil.JSONParameter("quux"),
				},
			},
			QArg: "foo.bar",
			Opts: []evaluate.Option{
				evaluate.WithParameterTypeResolver(resolve.NewMemoryParameterTypeResolver(
					map[string]interface{}{"quux": "baz", "grault": "garply"},
				)),
			},
			DefaultArg: testutil.JSONParameter("grault"),
			Expected:   "garply",
		},
		{
			Name: "path is not resolvable (using parameters)",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg:               "foo.bar",
			ExpectedIncomplete: true,
		},
		{
			Name: "path is not resolvable (using parameters) with default",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg:               "foo.bar",
			DefaultArg:         42,
			ExpectedIncomplete: true,
		},
		{
			Name: "path is resolvable (using parameters) but object is not completely resolvable",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
					"baz": "ok",
				},
			},
			QArg:       "foo.baz",
			DefaultArg: "ok",
			Expected:   "ok",
		},
		{
			Name: "query is not resolvable",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:               testutil.JSONParameter("wut"),
			ExpectedIncomplete: true,
		},
		{
			Name: "default is not resolvable",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:               "foo.quux",
			DefaultArg:         testutil.JSONParameter("wut"),
			ExpectedIncomplete: true,
		},
		{
			Name:       "object is unsupported type (nil)",
			ObjectArg:  nil,
			QArg:       "foo.bar",
			DefaultArg: "bar",
			ExpectedPositionalError: &fn.PositionalArgError{
				Arg: 1,
				Cause: &evaluate.PathEvaluationError{
					Path:  "foo",
					Cause: &jsonpath.UnsupportedValueTypeError{Value: nil},
				},
			},
			ExpectedKeywordError: &fn.KeywordArgError{
				Arg: "object",
				Cause: &evaluate.PathEvaluationError{
					Path:  "foo",
					Cause: &jsonpath.UnsupportedValueTypeError{Value: nil},
				},
			},
		},
		{
			Name: "object is unsupported type (complex128)",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": cmplx.Sqrt(12i),
				},
			},
			QArg:       "foo.bar",
			DefaultArg: "bar",
			ExpectedPositionalError: &fn.PositionalArgError{
				Arg: 1,
				Cause: &evaluate.PathEvaluationError{
					Path: "foo",
					Cause: &evaluate.PathEvaluationError{
						Path: "bar",
						Cause: &evaluate.UnsupportedValueError{
							Type: reflect.TypeOf(complex128(1)),
						},
					},
				},
			},
			ExpectedKeywordError: &fn.KeywordArgError{
				Arg: "object",
				Cause: &evaluate.PathEvaluationError{
					Path: "foo",
					Cause: &evaluate.PathEvaluationError{
						Path: "bar",
						Cause: &evaluate.UnsupportedValueError{
							Type: reflect.TypeOf(complex128(1)),
						},
					},
				},
			},
		},
	}.RunAll(t)
}

func (tt test) Run(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("path")
	require.NoError(t, err)

	t.Run(tt.Name, func(t *testing.T) {
		ctx := context.Background()
		t.Run("positional", func(t *testing.T) {
			args := []model.Evaluable{
				evaluate.NewScopedEvaluator(tt.ObjectArg, tt.Opts...),
				evaluate.NewScopedEvaluator(tt.QArg, tt.Opts...),
			}
			if tt.DefaultArg != nil {
				args = append(args, evaluate.NewScopedEvaluator(tt.DefaultArg, tt.Opts...))
			}

			invoker, err := desc.PositionalInvoker(args)
			require.NoError(t, err)

			r, err := invoker.Invoke(ctx)
			if tt.ExpectedPositionalError != nil {
				require.Equal(t, tt.ExpectedPositionalError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.ExpectedIncomplete, !r.Complete())
				if !tt.ExpectedIncomplete {
					require.Equal(t, tt.Expected, r.Value)
				} else {
					expected := make([]interface{}, len(args))
					for i, arg := range args {
						r, err := arg.EvaluateAll(ctx)
						require.NoError(t, err)
						expected[i] = r.Value
					}
					require.Equal(t, expected, r.Value)
				}
			}
		})

		t.Run("keyword", func(t *testing.T) {
			args := map[string]model.Evaluable{
				"object": evaluate.NewScopedEvaluator(tt.ObjectArg, tt.Opts...),
				"query":  evaluate.NewScopedEvaluator(tt.QArg, tt.Opts...),
			}
			if tt.DefaultArg != nil {
				args["default"] = evaluate.NewScopedEvaluator(tt.DefaultArg, tt.Opts...)
			}

			invoker, err := desc.KeywordInvoker(args)
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			if tt.ExpectedKeywordError != nil {
				require.Equal(t, tt.ExpectedKeywordError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.ExpectedIncomplete, !r.Complete())
				if !tt.ExpectedIncomplete {
					require.Equal(t, tt.Expected, r.Value)
				} else {
					expected := make(map[string]interface{})
					for k, arg := range args {
						r, err := arg.EvaluateAll(ctx)
						require.NoError(t, err)
						expected[k] = r.Value
					}
					require.Equal(t, expected, r.Value)
				}
			}
		})
	})
}
