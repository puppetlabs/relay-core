package op

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

// ConnectionsManager is responsible for accessing the backend where secrets
// are stored and retrieving values for a given connection.
type ConnectionsManager interface {
	Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error)
}

// EncodingConnectionsManager wraps a ConnectionManager and decodes the
// Connection.Spec values using the default encoding/transfer library.
type EncodingConnectionsManager struct {
	delegate ConnectionsManager
}

func (m EncodingConnectionsManager) Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error) {
	conn, err := m.delegate.Get(ctx, typ, name)
	if err != nil {
		return nil, err
	}

	for k, v := range conn.Spec {
		decoded, derr := transfer.DecodeFromTransfer(v.(string))
		if derr != nil {
			return nil, errors.NewConnectionsValueDecodingError().WithCause(derr)
		}

		conn.Spec[k] = string(decoded)
	}

	return conn, nil
}

func NewEncodingConnectionManager(cm ConnectionsManager) *EncodingConnectionsManager {
	return &EncodingConnectionsManager{delegate: cm}
}
