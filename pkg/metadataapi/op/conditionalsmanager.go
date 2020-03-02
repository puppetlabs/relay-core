package op

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type ConditionalsManager interface {
	Get(ctx context.Context, taskID task.Hash) (parse.Tree, errors.Error)
}
