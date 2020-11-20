package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestConcat(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("concat")
	require.NoError(t, err)

	tests := []struct {
		Name     string
		Args     []model.Evaluable
		Expected interface{}
	}{
		{
			Name:     "empty",
			Args:     []model.Evaluable{},
			Expected: "",
		},
		{
			Name: "basic",
			Args: []model.Evaluable{
				model.StaticEvaluable("Hello, "),
				model.StaticEvaluable("world!"),
			},
			Expected: "Hello, world!",
		},
		{
			Name: "type conversion",
			Args: []model.Evaluable{
				model.StaticEvaluable("H"),
				model.StaticEvaluable(3),
				model.StaticEvaluable("llo, world!"),
			},
			Expected: "H3llo, world!",
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
