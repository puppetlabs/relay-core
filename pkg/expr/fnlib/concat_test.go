package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/stretchr/testify/require"
)

func TestConcat(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("concat")
	require.NoError(t, err)

	invoker, err := desc.PositionalInvoker([]interface{}{"Hello, ", "world!"})
	require.NoError(t, err)

	r, err := invoker.Invoke(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Hello, world!", r)
}
