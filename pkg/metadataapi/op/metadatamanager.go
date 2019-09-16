package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type MetadataManager interface {
	GetByIP(ctx context.Context, ip string) (*task.Metadata, errors.Error)
}
