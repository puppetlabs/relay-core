package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

// ConnectionsManager is responsible for accessing the backend where secrets
// are stored and retrieving values for a given connection.
type ConnectionsManager interface {
	Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error)
}
