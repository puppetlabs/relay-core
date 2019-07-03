// Package vault provides an implementation of the Secrets interface
// that will talk to vault and fetch secrets using a temporary JWT.
package vault

import (
	"context"
	"io/ioutil"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

type Vault struct {
	cfg     *Config
	api     *vaultapi.Client
	session *vaultLoggedInClient
}

func (v *Vault) Login(ctx context.Context) errors.Error {
	var token string

	if v.cfg.K8sServiceAccountTokenPath != "" {
		b, err := ioutil.ReadFile(v.cfg.K8sServiceAccountTokenPath)
		if err != nil {
			return errors.NewSecretsK8sServiceAccountTokenReadError().WithCause(err)
		}

		jwt := string(b)

		data := make(map[string]interface{})
		data["jwt"] = jwt
		data["role"] = v.cfg.Role
		secret, err := v.api.Logical().Write("auth/kubernetes/login", data)
		if err != nil {
			return errors.NewSecretsVaultLoginError().WithCause(err)
		}

		token = secret.Auth.ClientToken
	} else {
		token = v.cfg.Token
	}

	vc := &vaultLoggedInClient{
		client:       v.api,
		workflowName: v.cfg.WorkflowName,
		engineMount:  v.cfg.EngineMount,
		token:        token,
	}

	v.session = vc

	return nil
}

func (v *Vault) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	return v.session.get(ctx, key)
}

func NewVaultWithKubernetesAuth(cfg *Config) (*Vault, errors.Error) {
	vc, err := vaultapi.NewClient(&vaultapi.Config{Address: cfg.Addr})
	if err != nil {
		return nil, errors.NewSecretsVaultSetupError().WithCause(err)
	}

	return &Vault{
		cfg: cfg,
		api: vc,
	}, nil
}

type vaultLoggedInClient struct {
	client       *vaultapi.Client
	workflowName string
	token        string
	engineMount  string
}

// read is just a shortcut to the Vault client's Read method
func (v *vaultLoggedInClient) read(path string) (*vaultapi.Secret, error) {
	return v.client.Logical().Read(path)
}

// mountPath returns a vault-api style path to the secret
func (v *vaultLoggedInClient) mountPath(key string) string {
	return path.Join(v.engineMount, "data", "workflows", v.workflowName, key)
}

// extractValue fetches the secret value from the secretRef key (standard location for nebula
// secret values under a path when using vault)
func (v *vaultLoggedInClient) extractValue(sec *vaultapi.Secret) (string, errors.Error) {
	vaultData, _ := sec.Data["data"].(map[string]interface{})

	val, ok := vaultData["value"]
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
func (v *vaultLoggedInClient) get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
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
