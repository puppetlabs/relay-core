package vault

import (
	"errors"
	"fmt"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
)

const defaultEngineMount = "nebula"

// VaultAuth is a vault client that creates roles tied to kubernetes service accounts
// and attaches policies to those roles.
type VaultAuth struct {
	client      *vaultapi.Client
	engineMount string
}

// Address returns the http server address to the vault server
func (v VaultAuth) Address() string {
	return v.client.Address()
}

// WritePolicy creates a readonly policy granting an entity access to only
// their secrets under a given path using workflowID
func (v *VaultAuth) WritePolicy(policyName, workflowID string) error {
	if policyName == "" {
		return errors.New("policy cannot be blank")
	}

	if workflowID == "" {
		return errors.New("workflowID cannot be blank")
	}

	readOnlyPolicy := fmt.Sprintf(`path "%s/*" { capabilities = ["read", "list"] }`, v.buildMountPath(workflowID))

	err := v.client.Sys().PutPolicy(policyName, readOnlyPolicy)
	if err != nil {
		return err
	}

	return nil
}

// DeletePolicy deletes a policy named policyName.
func (v *VaultAuth) DeletePolicy(policyName string) error {
	if policyName == "" {
		return errors.New("namespace cannot be blank")
	}

	err := v.client.Sys().DeletePolicy(policyName)
	if err != nil {
		return err
	}

	return nil
}

// WriteRole takes a kubernetes namespace, service account name and policy name
// and asks vault to create an auth role with those parameters.
func (v *VaultAuth) WriteRole(namespace, serviceAccount, policy string) error {
	if namespace == "" {
		return errors.New("namespace cannot be blank")
	}

	if serviceAccount == "" {
		return errors.New("serviceAccount cannot be blank")
	}

	if policy == "" {
		return errors.New("policy cannot be blank")
	}

	data := make(map[string]interface{})
	data["bound_service_account_names"] = serviceAccount
	data["bound_service_account_namespaces"] = namespace
	data["policies"] = []string{policy}

	_, err := v.client.Logical().Write(v.roleMountPath(namespace), data)
	if err != nil {
		return err
	}

	return nil
}

// DeleteRole takes a kubernetes namespace and deletes the role that
// the service account is attached to.
func (v *VaultAuth) DeleteRole(namespace string) error {
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
	return path.Join("auth/kubernetes/role", namespace)
}

// NewVaultAuth takes a vault addr and auth token and returns a new
// VaultAuth instance.
func NewVaultAuth(addr, token, engineMount string) (*VaultAuth, error) {
	v, err := vaultapi.NewClient(&vaultapi.Config{Address: addr})
	if err != nil {
		return nil, err
	}

	v.SetToken(token)

	if engineMount == "" {
		engineMount = defaultEngineMount
	}

	return &VaultAuth{
		client:      v,
		engineMount: engineMount,
	}, nil
}
