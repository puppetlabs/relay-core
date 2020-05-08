package vault

import (
	"context"
	"fmt"
	"path"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultEngineMount   = "nebula"
	defaultAuthMountPath = "auth/kubernetes"
)

type accessGranter struct {
	client         *vaultapi.Client
	namespace      string
	serviceAccount string
	authMount      string
	dataMount      string
}

func (ag accessGranter) GrantAccessForPaths(_ context.Context, secretPaths map[string]string) (map[string]*secrets.AccessGrant, error) {
	grants := make(map[string]*secrets.AccessGrant)
	pol := []string{}
	polTemplate := `path "%s/*" { capabilities = ["read", "list"] }`

	for k, p := range secretPaths {
		scopedPath := vaultDataPath(ag.dataMount, p)
		pol = append(pol, fmt.Sprintf(polTemplate, scopedPath))
		grants[k] = &secrets.AccessGrant{
			BackendAddr: ag.client.Address(),
			MountPath:   ag.dataMount,
			ScopedPath:  p,
		}
	}

	finalPolicy := strings.Join(pol, "\n")

	err := ag.client.Sys().PutPolicy(ag.namespace, finalPolicy)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	data["bound_service_account_names"] = ag.serviceAccount
	data["bound_service_account_namespaces"] = ag.namespace
	data["policies"] = []string{ag.namespace}

	if _, err := ag.client.Logical().Write(vaultAuthPath(ag.authMount, ag.namespace), data); err != nil {
		return nil, err
	}

	return grants, nil
}

type accessRevoker struct {
	client         *vaultapi.Client
	namespace      string
	serviceAccount string
	authMount      string
}

func (ar accessRevoker) RevokeAllAccess(_ context.Context) error {
	err := ar.client.Sys().DeletePolicy(ar.namespace)
	if err != nil {
		return err
	}

	mount := vaultAuthPath(ar.authMount, ar.namespace)
	if _, err := ar.client.Logical().Delete(mount); err != nil {
		return err
	}

	return nil
}

// VaultAuth is a vault client that creates roles tied to kubernetes service accounts
// and attaches policies to those roles.
type VaultAuth struct {
	client        *vaultapi.Client
	authMountPath string
	engineMount   string
}

func (v VaultAuth) ServiceAccountAccessGranter(sa *corev1.ServiceAccount) (secrets.AccessGranter, error) {
	return accessGranter{
		client:         v.client,
		namespace:      sa.GetNamespace(),
		serviceAccount: sa.GetName(),
		authMount:      v.authMountPath,
	}, nil
}

func (v VaultAuth) ServiceAccountAccessRevoker(sa *corev1.ServiceAccount) (secrets.AccessRevoker, error) {
	return accessRevoker{
		client:         v.client,
		namespace:      sa.GetNamespace(),
		serviceAccount: sa.GetName(),
		authMount:      v.authMountPath,
	}, nil
}

// NewVaultAuth takes a vault Config and returns a new VaultAuth instance.
func NewVaultAuth(cfg *Config) (*VaultAuth, error) {
	v, err := vaultapi.NewClient(&vaultapi.Config{Address: cfg.Addr})
	if err != nil {
		return nil, err
	}

	v.SetToken(cfg.Token)

	engineMount := cfg.EngineMount

	if engineMount == "" {
		engineMount = defaultEngineMount
	}

	authMountPath := cfg.K8sAuthMountPath
	if authMountPath == "" {
		authMountPath = defaultAuthMountPath
	}

	return &VaultAuth{
		client:        v,
		authMountPath: authMountPath,
		engineMount:   engineMount,
	}, nil
}

func vaultDataPath(engineMount, tail string) string {
	return path.Join(engineMount, "data", tail)
}

func vaultAuthPath(authMount, tail string) string {
	return path.Join(authMount, "role", tail)
}
