package memory

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConditionalsManager struct {
	data map[string]string
}

func (cm *ConditionalsManager) GetByTaskID(ctx context.Context, taskID string) (string, errors.Error) {
	if _, ok := cm.data[taskID]; !ok {
		return "", errors.NewTaskConditionalsNotFoundForID(taskID)
	}

	return cm.data[taskID], nil
}

func New(conditionals map[string]string) *ConditionalsManager {
	return &ConditionalsManager{data: conditionals}
}
