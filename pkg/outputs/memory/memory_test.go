package memory

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryMetadataManager(t *testing.T) {
	om := &OutputsManager{}

	ctx := context.Background()

	err := om.Put(ctx, "test-task", "test-key", bytes.NewBufferString("test-value"))
	require.NoError(t, err)

	out, err := om.Get(ctx, "test-task", "test-key")
	require.NoError(t, err)
	require.Equal(t, "test-value", string(out.Value))

	out, err = om.Get(ctx, "test-task2", "test-key")
	require.Error(t, err)

	out, err = om.Get(ctx, "test-task", "test-key2")
	require.Error(t, err)
}
