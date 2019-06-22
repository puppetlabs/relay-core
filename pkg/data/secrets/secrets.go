// Package secrets provides an interface into the remote secret store
package secrets

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Store interface {
	GetScopedSession(workflowName, taskName, token string) (ScopedSession, errors.Error)
}

type ScopedSession interface {
	Get(ctx context.Context, key string) (*Secret, errors.Error)
}
