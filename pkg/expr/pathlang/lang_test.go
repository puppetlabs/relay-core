package pathlang_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/clock/k8sext"
	"github.com/puppetlabs/leg/timeutil/pkg/clockctx"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/pathlang"
	"github.com/stretchr/testify/require"
	k8sclock "k8s.io/apimachinery/pkg/util/clock"
)

func TestExpressions(t *testing.T) {
	ctx := context.Background()

	now := time.Date(2021, time.May, 31, 0, 0, 0, 0, time.UTC)
	fc := k8sclock.NewFakeClock(now)
	ctx = clockctx.WithClock(ctx, k8sext.NewClock(fc))

	input := map[string]interface{}{
		"a": []interface{}{1, 2, 3},
		"b": 10,
		"c": map[string]interface{}{
			"x": 1,
			"y": 2,
		},
	}

	fac := pathlang.NewFactory(pathlang.WithFunctionMap{Map: fnlib.Library()})

	tests := []struct {
		Name          string
		Expression    string
		Expected      interface{}
		ExpectedError string
	}{
		{
			Name:       "map path",
			Expression: "c.x",
			Expected:   1,
		},
		{
			Name:       "array path",
			Expression: "a[0]",
			Expected:   1,
		},
		{
			Name:          "invalid map path",
			Expression:    "a.x",
			ExpectedError: `path "a.x": unexpected string index "x" for slice, must be convertible to int: strconv.ParseInt: parsing "x": invalid syntax`,
		},
		{
			Name:       "root",
			Expression: "$",
			Expected:   input,
		},
		{
			Name:       "map path from root",
			Expression: "$.c.x",
			Expected:   1,
		},
		{
			Name:          "invalid character after root",
			Expression:    "$$",
			ExpectedError: "unexpected \"$\" while scanning",
		},
		{
			Name:          "invalid path",
			Expression:    "$.",
			ExpectedError: "unexpected",
		},
		{
			Name:       "simple arithmetic",
			Expression: "c.x + c.y",
			Expected:   float64(3),
		},
		{
			Name:       "parentheses",
			Expression: "(c.x + c.y) * a[1]",
			Expected:   float64(6),
		},
		{
			Name:       "map creation",
			Expression: `{'foo': a[0] + a[1], 'bar': a[1] + a[2]}`,
			Expected:   map[string]interface{}{"foo": float64(3), "bar": float64(5)},
		},
		{
			Name:       "array creation",
			Expression: `["foo", a[0], c.y]`,
			Expected:   []interface{}{"foo", 1, 2},
		},
		{
			Name:       "function call with no arguments",
			Expression: "now()",
			Expected:   now,
		},
		{
			Name:       "function call with one positional argument",
			Expression: `jsonUnmarshal('{"foo": "bar"}')`,
			Expected:   map[string]interface{}{"foo": "bar"},
		},
		{
			Name:       "function call with more than one positional argument",
			Expression: "concat('a', b, 'c')",
			Expected:   "a10c",
		},
		{
			Name:       "function call with one keyword argument",
			Expression: "merge(objects: [{'a': 5}, {'b': b}])",
			Expected:   map[string]interface{}{"a": float64(5), "b": 10},
		},
		{
			Name:       "function call with multiple keyword arguments",
			Expression: "path(object: $, query: 'c.y')",
			Expected:   2,
		},
		{
			Name:       "call stack",
			Expression: "path(merge({'a': 5}, {'b': b}), 'b')",
			Expected:   10,
		},
		{
			Name:       "pipe",
			Expression: `jsonUnmarshal('{"x": {"y": "z"}}') |> x.y`,
			Expected:   "z",
		},
		{
			Name:       "pipe scope",
			Expression: `b * (jsonUnmarshal('{"x": 20}') |> x) + b`,
			Expected:   float64(210),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Run("expression", func(t *testing.T) {
				var u model.Unresolvable

				r, err := fac.Expression(&u).EvaluateWithContext(ctx, test.Expression, input)
				if test.ExpectedError != "" {
					require.NotNil(t, err)
					require.Contains(t, err.Error(), test.ExpectedError)
				} else {
					require.NoError(t, err)
					require.NoError(t, u.AsError())
					require.Equal(t, test.Expected, r)
				}
			})

			t.Run("template", func(t *testing.T) {
				var u model.Unresolvable

				r, err := fac.Template(&u).EvaluateWithContext(ctx, `${`+test.Expression+`}`, input)
				if test.ExpectedError != "" {
					require.NotNil(t, err)
					require.Contains(t, err.Error(), test.ExpectedError)
				} else {
					require.NoError(t, err)
					require.NoError(t, u.AsError())
					require.Equal(t, test.Expected, r)
				}
			})
		})
	}
}
