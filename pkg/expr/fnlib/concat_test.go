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
		Args     []interface{}
		Expected interface{}
	}{
		{
			Name:     "empty",
			Expected: "",
		},
		{
			Name:     "basic",
			Args:     []interface{}{"Hello, ", "world!"},
			Expected: "Hello, world!",
		},
		{
			Name:     "type conversion",
			Args:     []interface{}{"H", 3, "llo, world!"},
			Expected: "H3llo, world!",
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
