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
		Args     []interface{}
		Expected interface{}
	}{
		{
			Name:     "null then values",
			Args:     []interface{}{nil, 3, 4},
			Expected: 3,
		},
		{
			Name: "unresolvable then values",
			Args: []interface{}{
				model.StaticExpandable(nil, model.Unresolvable{
					Parameters: []model.UnresolvableParameter{
						{Name: "foo"},
					},
				}),
				3,
				4,
			},
			Expected: 3,
		},
		{
			Name:     "values first",
			Args:     []interface{}{1, nil},
			Expected: 1,
		},
		{
			Name:     "no arguments",
			Expected: nil,
		},
		{
			Name: "no unresolvable or non-null values",
			Args: []interface{}{
				nil,
				model.StaticExpandable(nil, model.Unresolvable{
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
			invoker, err := desc.PositionalInvoker(model.DefaultEvaluator, test.Args)
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			require.NoError(t, err)
			require.True(t, r.Complete())
			require.Equal(t, test.Expected, r.Value)
		})
	}
}
