package op

import (
	"context"
	"crypto/sha1"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
)

type OutputsManager interface {
	Get(ctx context.Context, taskName, key string) (*outputs.Output, errors.Error)
	Put(ctx context.Context, taskHash [sha1.Size]byte, key string, value transfer.JSONInterface) errors.Error
}
