package v1

import (
	"testing"

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
		},
	}

	manifest, err := mapper.ToRuntimeObjectsManifest(tenant.Tenant, source)
	require.NoError(t, err)

	require.NotNil(t, manifest.WebhookTrigger)
	require.Equal(t, tenant.Tenant.GetNamespace(), manifest.WebhookTrigger.GetNamespace())
}
