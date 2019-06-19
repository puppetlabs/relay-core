// Package vault provides an implementation of the Secrets interface
// that will talk to vault and fetch secrets using a temporary JWT.
package vault

import (
	"context"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Vault struct {
	addr string
	api  *vaultapi.Client
}

func (v *Vault) GetScopedSession(gid string, jwt string) (secrets.ScopedSession, errors.Error) {
	return &vaultSession{
		client: v,
		gid:    gid,
		jwt:    jwt,
	}, nil
}

func NewStore(addr string) (*Vault, errors.Error) {
	v, err := vaultapi.NewClient(vaultapi.Config{Address: addr})
	if err != nil {
		return errors.NewSecretsVaultSetupError().WithCause(err)
	}

	return &Vault{
		addr: addr,
	}, nil
}

type vaultScopedSession struct {
	client *Vault
	gid    string
	jwt    string
}

func (v *vaultScopedSession) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {

	return nil, nil
}
