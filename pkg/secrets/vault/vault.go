// Package vault provides an implementation of the Secrets interface
// that will talk to vault and fetch secrets using a temporary JWT.
package vault

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
)

type Vault struct {
	cfg     *Config
	session *vaultLoggedInClient
}

func (v *Vault) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	return v.session.get(ctx, key)
}

func NewVaultWithKubernetesAuth(ctx context.Context, cfg *Config) (*Vault, errors.Error) {
	session, err := newVaultLoggedInClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Vault{
		cfg:     cfg,
		session: session,
	}, nil
}

type vaultLoggedInClient struct {
	client *vaultapi.Client
	cfg    *Config
	logger logging.Logger
	// renewFunc takes a vault client and attempts to renew the auth token lease.
	// If the lease cannot be renewed, or an error occurres, then it will not restart
	// the renewFunc goroutine and then next attempt to use the token will fail.
	renewFunc func(context.Context, *vaultLoggedInClient)
}

// read is just a shortcut to the Vault client's Read method
func (v *vaultLoggedInClient) read(path string) (*vaultapi.Secret, error) {
	return v.client.Logical().Read(path)
}

// mountPath returns a vault-api style path to the secret
func (v *vaultLoggedInClient) mountPath(key string) string {
	return path.Join(v.cfg.ScopedSecretsPath, key)
}

// extractValue fetches the secret value from the secretRef key (standard location for nebula
// secret values under a path when using vault). Secret values must always be strings. Any special
// type handling should be processed at a higher level when decoding the string.
func (v *vaultLoggedInClient) extractValue(sec *vaultapi.Secret) (string, errors.Error) {
	vaultData, _ := sec.Data["data"].(map[string]interface{})

	val, ok := vaultData["value"]
	if !ok {
		return "", errors.NewSecretsMissingSecretRef().Bug()
	}

	ret, ok := val.(string)
	if !ok {
		return "", errors.NewSecretsMalformedValue()
	}

	return ret, nil
}

func (v *vaultLoggedInClient) login(ctx context.Context) errors.Error {
	var token string

	if v.cfg.Token != "" {
		token = v.cfg.Token
	} else if v.cfg.K8sServiceAccountTokenPath != "" {
		b, err := ioutil.ReadFile(v.cfg.K8sServiceAccountTokenPath)
		if err != nil {
			return errors.NewSecretsK8sServiceAccountTokenReadError().WithCause(err)
		}

		jwt := string(b)

		data := map[string]interface{}{
			"jwt":  jwt,
			"role": v.cfg.Role,
		}

		mountPath := defaultAuthMountPath
		if v.cfg.K8sAuthMountPath != "" {
			mountPath = v.cfg.K8sAuthMountPath
		}

		secret, err := v.client.Logical().Write(fmt.Sprintf("%s/login", mountPath), data)
		if err != nil {
			return errors.NewSecretsVaultLoginError().WithCause(err)
		}

		token = secret.Auth.ClientToken
	} else {
		return errors.NewSecretsVaultAuthenticationNotConfiguredError()
	}

	v.client.SetToken(token)

	tok, err := v.client.Auth().Token().LookupSelf()
	if err != nil {
		return errors.NewSecretsVaultTokenLookupError().WithCause(err)
	}

	// if our token is a root token, then it's "infinitely" usable, so we just skip the
	// renewal logic.
	if tok.Data["display_name"] != nil {
		name, ok := tok.Data["display_name"].(string)
		if ok && name == "root" {
			return nil
		}
	}

	renewable, err := tok.TokenIsRenewable()
	if err != nil {
		return errors.NewSecretsVaultTokenLookupError().WithCause(err)
	}

	if !renewable {
		return nil
	}

	ttl, err := tok.TokenTTL()
	if err != nil {
		return errors.NewSecretsVaultTokenLookupError().WithCause(err)
	}

	if ttl < time.Second*30 {
		v.logger.Warn("short vault token TTLs can lead to a renewal race condition where the token won't get renewed in time.", "ttl", ttl)
		v.logger.Warn("consider using a token with a lease greater than 1m")
	}

	go delayedTokenRenewal(ctx, v, ttl)

	return nil
}

// Get retrieves key from Vault using the session scoped workflow parameters and the
// temporary token.
func (v *vaultLoggedInClient) get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
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

func newVaultLoggedInClient(ctx context.Context, cfg *Config) (*vaultLoggedInClient, errors.Error) {
	c, err := vaultapi.NewClient(&vaultapi.Config{Address: cfg.Addr})
	if err != nil {
		return nil, errors.NewSecretsVaultSetupError().WithCause(err)
	}

	vlc := &vaultLoggedInClient{
		client: c,
		cfg:    cfg,
		logger: cfg.Logger.At("vault"),
	}

	if err := vlc.login(ctx); err != nil {
		return nil, err
	}

	return vlc, nil
}

func delayedTokenRenewal(ctx context.Context, v *vaultLoggedInClient, ttl time.Duration) {
	delay := time.Duration(float64(ttl.Nanoseconds()) * 2 / 3)
	timer := time.NewTimer(delay)

	for {
		select {
		case <-timer.C:
			_, err := v.client.Auth().Token().RenewSelf(0)
			if err != nil {
				v.logger.Info("could not renew the vault token", "error", err)

				return
			}

			v.logger.Info("vault token renewed successfully")

			timer.Reset(delay)
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}

			return
		}
	}
}
