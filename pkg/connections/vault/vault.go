package vault

import (
	"context"
	"path"

	"github.com/puppetlabs/nebula-tasks/pkg/connections"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
)

// ConnectionsManager uses the vault secrets backend to get connection data.
type ConnectionsManager struct {
	v *vault.Vault
}

// Get makes a call to vault's GetAll method to return a list of secret
// key/value pairs. The lookup key is the connection type/name. If an error is
// returned from vault, it will check to see if it's a not found error
// and then convert it to a ConnectionNotFoundError. Otherwise the error
// is converted to ConnectionGetError.
func (cm *ConnectionsManager) Get(ctx context.Context, typ, name string) (*connections.Connection, errors.Error) {
	resp, err := cm.v.GetAll(ctx, path.Join(typ, name))
	if err != nil {
		if errors.IsSecretsKeyNotFound(err) {
			return nil, errors.NewConnectionsTypeNameNotFound(typ, name).WithCause(err)
		}

		return nil, errors.NewConnectionsGetError().WithCause(err).Bug()
	}

	conn := &connections.Connection{Spec: make(map[string]string)}

	for _, sec := range resp {
		conn.Spec[sec.Key] = sec.Value
	}

	return conn, nil
}

// New takes a configured vault client and returns a new ConnectionsManager
func New(v *vault.Vault) *ConnectionsManager {
	return &ConnectionsManager{v: v}
}
