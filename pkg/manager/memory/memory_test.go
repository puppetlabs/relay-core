package memory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/stretchr/testify/require"
)

func TestMemoryMetadataManager(t *testing.T) {
	om := &outputs.MemoryManager{}

	ctx := context.Background()

	key := "test-key"
	value := "test-value"
	stepName := "test-task"
	run := uuid.New().String()

	thisTask := &task.Task{
		Run:  run,
		Name: stepName,
	}
	md := thisTask.TaskMetadata()

	err := om.Put(ctx, md, key, transfer.JSONInterface{Data: value})
	require.NoError(t, err)

	out, err := om.Get(ctx, md, stepName, key)
	require.NoError(t, err)
	require.Equal(t, value, out.Value.Data)

	out, err = om.Get(ctx, md, uuid.New().String(), key)
	require.Error(t, err)

	out, err = om.Get(ctx, md, stepName, uuid.New().String())
	require.Error(t, err)
}
