package vault

import (
	"context"
	"errors"
	"fmt"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
)

const (
	defaultEngineMount   = "nebula"
	defaultAuthMountPath = "auth/kubernetes"
)

// VaultAuth is a vault client that creates roles tied to kubernetes service accounts
// and attaches policies to those roles.
type VaultAuth struct {
	client        *vaultapi.Client
	authMountPath string
	engineMount   string
}

// GrantScopedAccess grants pods in namespace with serviceAccount attached access to secrets scoped under
// workflowID.
func (v VaultAuth) GrantScopedAccess(_ context.Context, workflowID, namespace, serviceAccount string) (*secrets.AccessGrant, error) {
	if err := v.writePolicy(namespace, workflowID); err != nil {
		return nil, err
	}

	if err := v.writeRole(namespace, serviceAccount); err != nil {
		return nil, err
	}

	return &secrets.AccessGrant{
		BackendAddr: v.client.Address(),
		ScopedPath:  v.buildMountPath(workflowID),
	}, nil
}

// RevokeScopedAccess deletes roles and policies related to namespace.
func (v VaultAuth) RevokeScopedAccess(_ context.Context, namespace string) error {
	if err := v.deletePolicy(namespace); err != nil {
		return err
	}

	if err := v.deleteRole(namespace); err != nil {
		return err
	}

	return nil
}

// writePolicy creates a readonly policy granting an entity access to only
// their secrets under a given path using workflowID
func (v *VaultAuth) writePolicy(namespace, workflowID string) error {
	if namespace == "" {
		return errors.New("policy cannot be blank")
	}

	if workflowID == "" {
		return errors.New("workflowID cannot be blank")
	}

	readOnlyPolicy := fmt.Sprintf(`path "%s/*" { capabilities = ["read", "list"] }`, v.buildMountPath(workflowID))

	err := v.client.Sys().PutPolicy(namespace, readOnlyPolicy)
	if err != nil {
		return err
	}

	return nil
}

// deletePolicy deletes a policy for namespace.
func (v *VaultAuth) deletePolicy(namespace string) error {
	if namespace == "" {
		return errors.New("namespace cannot be blank")
	}

	err := v.client.Sys().DeletePolicy(namespace)
	if err != nil {
		return err
	}

	return nil
}

// writeRole takes a kubernetes namespace, service account name
// and asks vault to create an auth role with those parameters.
// It uses the namespace as the policy name.
func (v *VaultAuth) writeRole(namespace, serviceAccount string) error {
	if namespace == "" {
		return errors.New("namespace cannot be blank")
	}

	if serviceAccount == "" {
		return errors.New("serviceAccount cannot be blank")
	}

	data := make(map[string]interface{})
	data["bound_service_account_names"] = serviceAccount
	data["bound_service_account_namespaces"] = namespace
	data["policies"] = []string{namespace}

	_, err := v.client.Logical().Write(v.roleMountPath(namespace), data)
	if err != nil {
		return err
	}

	return nil
}

// deleteRole takes a kubernetes namespace and deletes the role that
// the service account is attached to.
func (v *VaultAuth) deleteRole(namespace string) error {
	if namespace == "" {
		return errors.New("namespace cannot be blank")
	}

	_, err := v.client.Logical().Delete(v.roleMountPath(namespace))
	if err != nil {
		return err
	}

	return nil
}

// buildMountPath takes a workflowID and returns a mount path with a trailing /
func (v *VaultAuth) buildMountPath(workflowID string) string {
	return path.Join(v.engineMount, "data", "workflows", workflowID, "/")
}

// roleMountPath takes a namespace and returns a role path using the vault
// k8s auth path prefix
func (v *VaultAuth) roleMountPath(namespace string) string {
	return path.Join(fmt.Sprintf("%s/role", v.authMountPath), namespace)
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
