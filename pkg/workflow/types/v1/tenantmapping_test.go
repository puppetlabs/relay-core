package v1

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTenantEngineMapping(t *testing.T) {
	u, err := url.Parse("https://example.com/events")
	require.NoError(t, err)

	mapper := NewDefaultTenantEngineMapper(
		WithNamespaceTenantOption("tenant-namespace"),
		WithNameTenantOption("test-tenant"),
		WithIDTenantOption("test-123"),
		WithWorkflowIDTenantOption("workflow-123"),
		WithEventURLTenantOption(u),
		WithTokenSecretNameTenantOption("test-token-secret"),
	)

	manifest, err := mapper.ToRuntimeObjectsManifest()
	require.NoError(t, err)

	require.NotNil(t, manifest.Namespace)
	require.NotNil(t, manifest.Tenant)

	require.Equal(t, "test-tenant", manifest.Tenant.GetName())

	require.NotNil(t, manifest.Tenant.Spec.TriggerEventSink)
	require.Equal(t, u.String(), manifest.Tenant.Spec.TriggerEventSink.API.URL)
	require.Equal(t, "test-token-secret", manifest.Tenant.Spec.TriggerEventSink.API.TokenFrom.SecretKeyRef.LocalObjectReference.Name)

	require.NoError(t, json.NewEncoder(ioutil.Discard).Encode(manifest.Namespace))
	require.NoError(t, json.NewEncoder(ioutil.Discard).Encode(manifest.Tenant))
}
