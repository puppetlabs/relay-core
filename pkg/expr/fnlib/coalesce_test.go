package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestCoalesce(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("coalesce")
	require.NoError(t, err)

	tests := []struct {
		Name     string
		Args     []model.Evaluable
		Expected interface{}
	}{
		{
			Name: "null then values",
			Args: []model.Evaluable{
				model.StaticEvaluable(nil),
				model.StaticEvaluable(3),
				model.StaticEvaluable(4),
			},
			Expected: 3,
		},
		{
			Name: "unresolvable then values",
			Args: []model.Evaluable{
				model.UnresolvableEvaluable(model.Unresolvable{
					Parameters: []model.UnresolvableParameter{
						{Name: "foo"},
					},
				}),
				model.StaticEvaluable(3),
				model.StaticEvaluable(4),
			},
			Expected: 3,
		},
		{
			Name: "values first",
			Args: []model.Evaluable{
				model.StaticEvaluable(1),
				model.StaticEvaluable(nil),
			},
			Expected: 1,
		},
		{
			Name:     "no arguments",
			Args:     []model.Evaluable{},
			Expected: nil,
		},
		{
			Name: "no unresolvable or non-null values",
			Args: []model.Evaluable{
				model.StaticEvaluable(nil),
				model.UnresolvableEvaluable(model.Unresolvable{
					Parameters: []model.UnresolvableParameter{
						{Name: "foo"},
					},
				}),
			},
			Expected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			invoker, err := desc.PositionalInvoker(test.Args)
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			require.NoError(t, err)
			require.True(t, r.Complete())
			require.Equal(t, test.Expected, r.Value)
		})
	}
}
