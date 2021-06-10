package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestJSONUnmarshal(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("jsonUnmarshal")
	require.NoError(t, err)

	invoker, err := desc.PositionalInvoker(model.DefaultEvaluator, []interface{}{`{"foo": "bar"}`})
	require.NoError(t, err)

	r, err := invoker.Invoke(context.Background())
	require.NoError(t, err)
	require.True(t, r.Complete())
	require.Equal(t, map[string]interface{}{"foo": "bar"}, r.Value)
}
