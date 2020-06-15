package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/fnlib"
	"github.com/stretchr/testify/require"
)

func TestAppend(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("append")
	require.NoError(t, err)

	invoker, err := desc.PositionalInvoker([]interface{}{
		[]interface{}{1, 2},
		3,
		4,
	})
	require.NoError(t, err)

	r, err := invoker.Invoke(context.Background())
	require.NoError(t, err)
	require.Equal(t, []interface{}{1, 2, 3, 4}, r)
}
