package op

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type OutputsManager interface {
	Get(ctx context.Context, metadata *task.Metadata, stepName, key string) (*outputs.Output, errors.Error)
	Put(ctx context.Context, metadata *task.Metadata, key string, value transfer.JSONInterface) errors.Error
}
