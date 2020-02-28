package memory

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type ConditionalsManager struct {
	data map[string]string
}

func (cm *ConditionalsManager) Get(ctx context.Context, taskHash task.Hash) (parse.Tree, errors.Error) {
	taskHashKey := taskHash.HexEncoding()

	if _, ok := cm.data[taskHashKey]; !ok {
		return "", errors.NewTaskConditionalsNotFoundForID(taskHashKey)
	}

	tree, perr := parse.ParseJSONString(cm.data[taskHashKey])
	if perr != nil {
		return nil, errors.NewTaskConditionalsDecodingError().WithCause(perr)
	}

	return tree, nil
}

func New(conditionals map[string]string) *ConditionalsManager {
	return &ConditionalsManager{data: conditionals}
}
