package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type ConditionalsManager interface {
	GetByTaskID(ctx context.Context, taskID string) (string, errors.Error)
}
