package task_test

import (
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	name := "my-test-step"

	h1 := task.HashFromName(name)
	h2, err := task.HashFromID(h1.HexEncoding())
	require.NoError(t, err)
	require.Equal(t, h1, h2)
}
