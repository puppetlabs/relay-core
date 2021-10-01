package authenticate

import (
	"strings"

	"github.com/puppetlabs/leg/jsonutil/pkg/types"
	"github.com/puppetlabs/relay-core/pkg/model"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Claims struct {
	*jwt.Claims `json:",inline"`

	KubernetesNamespaceName       string `json:"k8s.io/namespace-name,omitempty"`
	KubernetesNamespaceUID        string `json:"k8s.io/namespace-uid,omitempty"`
	KubernetesServiceAccountToken string `json:"k8s.io/service-account-token,omitempty"`

	// RelayDomainID represents a holder of tenants with high-level
	// configuration like connections. In our SaaS service, domains correspond
	// to accounts.
	RelayDomainID string `json:"relay.sh/domain-id,omitempty"`

	// RelayTenantID represents the root of configuration for secrets, etc. In
	// our SaaS service, tenants correspond to workflows.
	RelayTenantID string `json:"relay.sh/tenant-id,omitempty"`

	RelayName  string `json:"relay.sh/name,omitempty"`
	RelayRunID string `json:"relay.sh/run-id,omitempty"`

	RelayKubernetesImmutableConfigMapName string `json:"relay.sh/k8s/immutable-config-map-name,omitempty"`
	RelayKubernetesMutableConfigMapName   string `json:"relay.sh/k8s/mutable-config-map-name,omitempty"`

	RelayVaultEnginePath     string `json:"relay.sh/vault/engine-path,omitempty"`
	RelayVaultSecretPath     string `json:"relay.sh/vault/secret-path,omitempty"`
	RelayVaultConnectionPath string `json:"relay.sh/vault/connection-path,omitempty"`

	RelayEventAPIURL   *types.URL `json:"relay.sh/event/api/url,omitempty"`
	RelayEventAPIToken string     `json:"relay.sh/event/api/token,omitempty"`

	RelayWorkflowExecutionAPIURL   *types.URL `json:"relay.sh/workflow-execution/api/url,omitempty"`
	RelayWorkflowExecutionAPIToken string     `json:"relay.sh/workflow-execution/api/token,omitempty"`
}

func (c *Claims) Action() model.Action {
	parts := strings.SplitN(c.Subject, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	var action model.Action

	switch parts[0] {
	case model.ActionTypeStep.Plural:
		if c.RelayRunID == "" || c.RelayName == "" {
			return nil
		}

		action = &model.Step{
			Run:  model.Run{ID: c.RelayRunID},
			Name: c.RelayName,
		}
	case model.ActionTypeTrigger.Plural:
		if c.RelayName == "" {
			return nil
		}

		action = &model.Trigger{
			Name: c.RelayName,
		}
	default:
		return nil
	}

	// This is a sanity check to make sure we read all information correctly
	// when reconstructing the action.
	if action.Hash().HexEncoding() != parts[1] {
		return nil
	}

	return action
}
