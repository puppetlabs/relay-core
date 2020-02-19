package memory

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConditionalsManager struct {
	data map[string]string
}

func (m *ConditionalsManager) GetByTaskID(ctx context.Context, taskID string) (string, errors.Error) {
	return "", nil
}
