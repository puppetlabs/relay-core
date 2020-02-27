package op

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type OutputsManager interface {
	Get(ctx context.Context, taskName, key string) (*outputs.Output, errors.Error)
	Put(ctx context.Context, taskHash task.Hash, key string, value transfer.JSONInterface) errors.Error
}
