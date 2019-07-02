package vault

import (
	"path"

	vaultapi "github.com/hashicorp/vault/api"
)

type VaultAuth struct {
	client *vaultapi.Client
}

func (v *VaultAuth) WriteServiceAccount(namespace, serviceAccount, policy string) error {
	p = path.Join("auth/kubernetes/role", namespace)

	return nil
}

func (v *VaultAuth) DeleteServiceAccount() error {
	return nil
}

func NewVaultAuth(addr string, token string) (*VaultAuth, error) {
	v, err := vaultapi.NewClient(&vaultapi.Config{Address: addr})
	if err != nil {
		return nil, err
	}

	v.SetToken(token)

	return &VaultAuth{client: v}, nil
}
