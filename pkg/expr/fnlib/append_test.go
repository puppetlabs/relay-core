package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestAppend(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("append")
	require.NoError(t, err)

	invoker, err := desc.PositionalInvoker([]model.Evaluable{
		model.StaticEvaluable([]interface{}{1, 2}),
		model.StaticEvaluable(3),
		model.StaticEvaluable(4),
	})
	require.NoError(t, err)

	r, err := invoker.Invoke(context.Background())
	require.NoError(t, err)
	require.True(t, r.Complete())
	require.Equal(t, []interface{}{1, 2, 3, 4}, r.Value)
}
