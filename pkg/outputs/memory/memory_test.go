package memory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/stretchr/testify/require"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

func TestMemoryMetadataManager(t *testing.T) {
	om := &OutputsManager{}

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
