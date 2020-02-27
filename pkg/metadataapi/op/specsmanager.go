package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type SpecsManager interface {
	Get(ctx context.Context, taskID task.Hash) (string, errors.Error)
}
