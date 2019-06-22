// Package vault provides an implementation of the Secrets interface
// that will talk to vault and fetch secrets using a temporary JWT.
package vault

import (
	"context"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Vault struct {
	addr string
	api  *vaultapi.Client
}

func (v *Vault) GetScopedSession(workflowName, taskName, token string) (secrets.ScopedSession, errors.Error) {
	clone, err := v.api.Clone()
	if err != nil {
		return nil, errors.NewSecretsSessionSetupError().WithCause(err)
	}

	return &vaultScopedSession{
		client:       clone,
		workflowName: workflowName,
		taskName:     taskName,
		token:        token,
	}, nil
}

func New(addr string) (*Vault, errors.Error) {
	v, err := vaultapi.NewClient(&vaultapi.Config{Address: addr})
	if err != nil {
		return nil, errors.NewSecretsVaultSetupError().WithCause(err)
	}

	return &Vault{
		addr: addr,
		api:  v,
	}, nil
}

type vaultScopedSession struct {
	client       *vaultapi.Client
	workflowName string
	taskName     string
	token        string
}

// read is just a shortcut to the Vault client's Read method
func (v *vaultScopedSession) read(path string) (*vaultapi.Secret, error) {
	return v.client.Logical().Read(path)
}

// mountPath returns a vault-api style path to the secret
func (v *vaultScopedSession) mountPath(key string) string {
	return path.Join(v.workflowName, "data", v.taskName, key)
}

// extractValue fetches the secret value from the secretRef key (standard location for nebula
// secret values under a path when using vault)
func (v *vaultScopedSession) extractValue(sec *vaultapi.Secret) (string, errors.Error) {
	vaultData, _ := sec.Data["data"].(map[string]interface{})

	val, ok := vaultData["secretRef"]
	if !ok {
		return "", errors.NewSecretsMissingSecretRef().Bug()
	}

	// TODO: for now we only support string values for secrets.
	// A couple reasons for this: supporting other types is very hard
	// and it's unclear how we should design for that right now, and
	// if we intend on utilizing environment variables we will want strings
	// because... bash.
	ret, ok := val.(string)
	if !ok {
		return "", errors.NewSecretsMalformedValue()
	}

	return ret, nil
}

// Get retrieves key from Vault using the session scoped workflow parameters and the
// temporary token.
func (v *vaultScopedSession) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	v.client.SetToken(v.token)

	sec, err := v.read(v.mountPath(key))
	if err != nil {
		return nil, errors.NewSecretsGetError(key).WithCause(err).Bug()
	}

	if sec == nil {
		return nil, errors.NewSecretsKeyNotFound(key)
	}

	val, err := v.extractValue(sec)
	if err != nil {
		return nil, errors.NewSecretsGetError(key).WithCause(err).Bug()
	}

	return &secrets.Secret{
		Key:   key,
		Value: val,
	}, nil
}
