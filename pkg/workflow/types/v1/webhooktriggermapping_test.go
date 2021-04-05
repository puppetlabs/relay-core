package v1

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	"github.com/stretchr/testify/require"
)

func TestWebhookTriggerMapping(t *testing.T) {
	tenantMapper := NewDefaultTenantEngineMapper(
		WithIDTenantOption("test-1234"),
		WithNameTenantOption("test-name"),
		WithWorkflowIDTenantOption("workflow-1234"),
		WithNamespaceTenantOption("test-tenant"),
	)

	tenant, err := tenantMapper.ToRuntimeObjectsManifest()
	require.NoError(t, err)

	mapper := NewDefaultWebhookTriggerEngineMapper(
		WithIDWebhookTriggerOption("test-1234"),
		WithNameWebhookTriggerOption("test-webhook-trigger"),
		WithImageWebhookTriggerOption("test-image"),
	)

	source := &WebhookWorkflowTriggerSource{
		ContainerMixin: ContainerMixin{
			Image: "test-image:latest",
			Spec: ExpressionMap{
				"tag": serialize.JSONTree{Tree: "v1"},
			},
			Env: ExpressionMap{
				"CI":      serialize.JSONTree{Tree: true},
				"RETRIES": serialize.JSONTree{Tree: 3},
			},
		},
	}

	manifest, err := mapper.ToRuntimeObjectsManifest(tenant.Tenant, source)
	require.NoError(t, err)

	require.NotNil(t, manifest.WebhookTrigger)
	require.Equal(t, tenant.Tenant.GetNamespace(), manifest.WebhookTrigger.GetNamespace())

	require.Len(t, manifest.WebhookTrigger.Spec.Spec, 1)
	require.Len(t, manifest.WebhookTrigger.Spec.Env, 2)

	require.NoError(t, json.NewEncoder(ioutil.Discard).Encode(manifest.WebhookTrigger))
}
