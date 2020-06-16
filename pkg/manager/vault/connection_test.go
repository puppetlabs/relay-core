package vault_test

import (
	"context"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/manager/vault"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestConnectionManager(t *testing.T) {
	ctx := context.Background()

	testutil.WithVault(t, func(vcfg *testutil.Vault) {
		id := uuid.New().String()

		// Write data.
		attrs := map[string]interface{}{
			"foo": "bar",
			"baz": "quux",
		}
		for k, v := range attrs {
			_, err := vcfg.Client.Logical().Write(path.Join(vcfg.SecretsPath, "data", "foo", id, k), map[string]interface{}{
				"data": map[string]interface{}{
					"value": v,
				},
			})
			require.NoError(t, err)
		}

		// Write pointer.
		_, err := vcfg.Client.Logical().Write(path.Join(vcfg.SecretsPath, "data", "foo/some-type/test"), map[string]interface{}{
			"data": map[string]interface{}{
				"value": id,
			},
		})
		require.NoError(t, err)

		cm := vault.NewConnectionManager(vault.NewKVV2Client(vcfg.Client, vcfg.SecretsPath).In("foo"))

		conn, err := cm.Get(ctx, "some-type", "test")
		require.NoError(t, err)
		require.Equal(t, "some-type", conn.Type)
		require.Equal(t, "test", conn.Name)
		require.Equal(t, attrs, conn.Attributes)

		_, err = cm.Get(ctx, "some-other-type", "test")
		require.Equal(t, model.ErrNotFound, err)
	})
}
