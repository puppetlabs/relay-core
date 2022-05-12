package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/pathlang"
	"github.com/stretchr/testify/require"
)

func TestExists(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("exists")
	require.NoError(t, err)

	tests := []struct {
		Name          string
		Expr          string
		Param         any
		Expected      bool
		ExpectedError string
	}{
		{
			Name:     "basic",
			Expr:     "foo",
			Param:    map[string]any{"foo": "bar"},
			Expected: true,
		},
		{
			Name:     "missing key",
			Expr:     "foo.bar",
			Param:    map[string]any{"foo": map[string]any{"baz": "quux"}},
			Expected: false,
		},
		{
			Name:     "missing key at depth",
			Expr:     "foo.bar.baz",
			Param:    map[string]any{"foo": map[string]any{"baz": "quux"}},
			Expected: false,
		},
		{
			Name:     "index out of bounds",
			Expr:     "foo[1].bar",
			Param:    map[string]any{"foo": []any{map[string]any{"baz": "quux"}}},
			Expected: false,
		},
		{
			Name:          "invalid type",
			Expr:          "foo.bar",
			Param:         map[string]any{"foo": "bar"},
			ExpectedError: "unsupported value type string",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var u model.Unresolvable
			eval, err := pathlang.DefaultFactory.Expression(&u).NewEvaluable(test.Expr)
			require.NoError(t, err)

			invoker, err := desc.PositionalInvoker(model.DefaultEvaluator, []interface{}{model.EvalExpandable(eval, test.Param)})
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			if test.ExpectedError != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), test.ExpectedError)
			} else {
				require.NoError(t, err)

				r.Unresolvable.Extends(u)
				require.True(t, r.Complete())
				require.Equal(t, test.Expected, r.Value)
			}
		})
	}
}
