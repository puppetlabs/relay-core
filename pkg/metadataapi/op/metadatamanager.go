package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type MetadataManager interface {
	Get(context.Context) (*task.Metadata, errors.Error)
}
