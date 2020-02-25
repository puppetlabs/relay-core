package memory

import (
	"context"
	"crypto/sha1"
	"testing"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/stretchr/testify/require"
)

func TestMemoryMetadataManager(t *testing.T) {
	om := &OutputsManager{}

	ctx := context.Background()

	err := om.Put(ctx, sha1.Sum([]byte("test-task")), "test-key", transfer.JSONInterface{Data: "test-value"})
	require.NoError(t, err)

	out, err := om.Get(ctx, "test-task", "test-key")
	require.NoError(t, err)
	require.Equal(t, "test-value", out.Value.Data)

	out, err = om.Get(ctx, "test-task2", "test-key")
	require.Error(t, err)

	out, err = om.Get(ctx, "test-task", "test-key2")
	require.Error(t, err)
}
