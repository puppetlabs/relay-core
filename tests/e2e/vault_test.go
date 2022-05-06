package e2e_test

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/puppetlabs/leg/k8sutil/pkg/app/portforward"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/vaultutil/pkg/model"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Vault struct {
	client *api.Client
}

func (v *Vault) SetSecret(t *testing.T, ctx context.Context, tenantID, name, value string) {
	_, err := v.client.Logical().Write(path.Join(TestVaultEngineTenantPath, "data/workflows", tenantID, name), map[string]interface{}{
		"data": map[string]interface{}{
			"value": value,
		},
	})
	require.NoError(t, err)
}

func (v *Vault) SetConnection(t *testing.T, ctx context.Context, domainID, typ, name string, attrs map[string]string) {
	id := uuid.New().String()

	for k, vi := range attrs {
		_, err := v.client.Logical().Write(path.Join(TestVaultEngineTenantPath, "data/connections", domainID, id, k), map[string]interface{}{
			"data": map[string]interface{}{
				"value": vi,
			},
		})
		require.NoError(t, err)
	}

	_, err := v.client.Logical().Write(path.Join(TestVaultEngineTenantPath, "data/connections", domainID, typ, name), map[string]interface{}{
		"data": map[string]interface{}{
			"value": id,
		},
	})
	require.NoError(t, err)
}

func WithVault(t *testing.T, ctx context.Context, eit *EnvironmentInTest, fn func(v *Vault)) {
	secret := corev1obj.NewOpaqueSecret(client.ObjectKey{
		Namespace: TestSystemNamespace,
		Name:      TestVaultCredentialsSecretName,
	})
	_, err := lifecycle.NewRetryLoader(
		secret,
		func(ok bool, err error) (bool, error) {
			return ok && len(secret.Object.Data[model.VaultRootToken]) > 0, err
		},
	).Load(ctx, eit.ControllerClient)
	require.NoError(t, err)

	svc := corev1obj.NewService(client.ObjectKey{
		Namespace: TestSystemNamespace,
		Name:      TestVaultServiceName,
	})
	_, err = lifecycle.NewRetryLoader(svc, func(ok bool, err error) (bool, error) { return ok, err }).Load(ctx, eit.ControllerClient)
	require.NoError(t, err)

	require.NoError(t, portforward.ForwardService(ctx, eit.RESTConfig, svc, 8200, func(ctx context.Context, port uint16) error {
		cfg := api.DefaultConfig()
		cfg.Address = fmt.Sprintf("http://localhost:%d", port)

		cl, err := api.NewClient(cfg)
		require.NoError(t, err)
		cl.SetToken(string(secret.Object.Data[model.VaultRootToken]))

		fn(&Vault{client: cl})
		return nil
	}))
}
