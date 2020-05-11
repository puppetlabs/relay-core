package vault_test

import (
	"context"
	"path"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/manager/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestSecretManager(t *testing.T) {
	ctx := context.Background()

	testutil.WithVault(t, func(vcfg *testutil.Vault) {
		_, err := vcfg.Client.Logical().Write(path.Join(vcfg.SecretsPath, "data/foo/bar"), map[string]interface{}{
			"data": map[string]interface{}{
				"value": "baz",
			},
		})
		require.NoError(t, err)

		sm := vault.NewSecretManager(vcfg.Client, path.Join(vcfg.SecretsPath, "data/foo"))

		sec, err := sm.Get(ctx, "bar")
		require.NoError(t, err)
		require.Equal(t, "baz", sec.Value)

		_, err = sm.Get(ctx, "nonexistent")
		require.Equal(t, model.ErrNotFound, err)
	})
}
