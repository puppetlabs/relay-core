// Package secrets provides an interface into the remote secret store
package secrets

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Store interface {
	GetScopedSession(gid string, jwt string) (ScopedSession, errors.Error)
}

type ScopedSession interface {
	Get(ctx context.Context, key string) (*Secret, errors.Error)
}
