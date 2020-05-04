package vault_test

import (
	"context"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestSecretManager(t *testing.T) {
	ctx := context.Background()

	testutil.WithTestVaultClient(t, func(vc *vaultapi.Client) {
		require.NoError(t, vc.Sys().Mount("kv-secrets", &vaultapi.MountInput{
			Type: "kv-v2",
		}))

		_, err := vc.Logical().Write("kv-secrets/data/foo/bar", map[string]interface{}{
			"data": map[string]interface{}{
				"value": "baz",
			},
		})
		require.NoError(t, err)

		sm := vault.NewSecretManager(vc, "kv-secrets/data/foo")

		sec, err := sm.Get(ctx, "bar")
		require.NoError(t, err)
		require.Equal(t, "baz", sec.Value)

		_, err = sm.Get(ctx, "nonexistent")
		require.Equal(t, model.ErrNotFound, err)
	})
}
