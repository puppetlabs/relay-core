// Package secrets provides an interface into the remote secret store
package secrets

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Store interface {
	Login(ctx context.Context) errors.Error
	Get(ctx context.Context, key string) (*Secret, errors.Error)
}
