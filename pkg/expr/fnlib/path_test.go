package fnlib_test

import (
	"context"
	"testing"

	jsonpath "github.com/puppetlabs/paesslerag-jsonpath"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("path")
	require.NoError(t, err)

	tests := []struct {
		Name                    string
		ObjectArg               interface{}
		QArg                    string
		DefaultArg              interface{}
		Expected                interface{}
		ExpectedIncomplete      bool
		ExpectedPositionalError error
		ExpectedKeywordError    error
	}{
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
				Arg:   1,
				Cause: &jsonpath.UnknownKeyError{Key: "quux"},
			},
			ExpectedKeywordError: &fn.KeywordArgError{
				Arg:   "object",
				Cause: &jsonpath.UnknownKeyError{Key: "quux"},
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
			Name: "path is not resolvable",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg:               "foo.bar",
			ExpectedIncomplete: true,
		},
		{
			Name: "path is not resolvable with default",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
				},
			},
			QArg:       "foo.bar",
			DefaultArg: 42,
			Expected:   42,
		},
		{
			Name: "path is resolvable but object is not completely resolvable",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": testutil.JSONParameter("quux"),
					"baz": "ok",
				},
			},
			QArg:     "foo.baz",
			Expected: "ok",
		},
		{
			Name: "default is not resolvable (falls back to query error)",
			ObjectArg: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			QArg:       "foo.quux",
			DefaultArg: testutil.JSONParameter("wut"),
			ExpectedPositionalError: &fn.PositionalArgError{
				Arg:   1,
				Cause: &jsonpath.UnknownKeyError{Key: "quux"},
			},
			ExpectedKeywordError: &fn.KeywordArgError{
				Arg:   "object",
				Cause: &jsonpath.UnknownKeyError{Key: "quux"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Run("positional", func(t *testing.T) {
				args := []model.Evaluable{
					// Need a real evaluator to test pathing support.
					evaluate.NewScopedEvaluator(test.ObjectArg),
					model.StaticEvaluable(test.QArg),
				}
				if test.DefaultArg != nil {
					args = append(args, evaluate.NewScopedEvaluator(test.DefaultArg))
				}

				invoker, err := desc.PositionalInvoker(args)
				require.NoError(t, err)

				r, err := invoker.Invoke(context.Background())
				if test.ExpectedPositionalError != nil {
					require.Equal(t, test.ExpectedPositionalError, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, test.ExpectedIncomplete, !r.Complete())
					if !test.ExpectedIncomplete {
						require.Equal(t, test.Expected, r.Value)
					}
				}
			})

			t.Run("keyword", func(t *testing.T) {
				args := map[string]model.Evaluable{
					"object": evaluate.NewScopedEvaluator(test.ObjectArg),
					"query":  model.StaticEvaluable(test.QArg),
				}
				if test.DefaultArg != nil {
					args["default"] = evaluate.NewScopedEvaluator(test.DefaultArg)
				}

				invoker, err := desc.KeywordInvoker(args)
				require.NoError(t, err)

				r, err := invoker.Invoke(context.Background())
				if test.ExpectedKeywordError != nil {
					require.Equal(t, test.ExpectedKeywordError, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, test.ExpectedIncomplete, !r.Complete())
					if !test.ExpectedIncomplete {
						require.Equal(t, test.Expected, r.Value)
					}
				}
			})
		})
	}
}
