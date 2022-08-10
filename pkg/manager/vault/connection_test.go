package vault_test

import (
	"context"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
	"github.com/puppetlabs/relay-core/pkg/manager/vault"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestConnectionManager(t *testing.T) {
	ctx := context.Background()

	testutil.WithVaultClient(t, func(client *api.Client) {
		require.NoError(t, client.Sys().Mount("kv-test", &api.MountInput{
			Type: "kv-v2",
		}))

		id := uuid.New().String()

		// Write data.
		attrs := map[string]interface{}{
			"foo":  "bar",
			"baz":  "quux",
			"flub": "wh\nup",
		}
		for k, v := range attrs {
			_, err := client.Logical().Write(path.Join("kv-test/data/foo", id, k), map[string]interface{}{
				"data": map[string]interface{}{
					"value": v,
				},
			})
			require.NoError(t, err)
		}

		// Write pointer.
		_, err := client.Logical().Write("kv-test/data/foo/some-type/test", map[string]interface{}{
			"data": map[string]interface{}{
				"value": id,
			},
		})
		require.NoError(t, err)

		cm := vault.NewConnectionManager(vault.NewKVV2Client(client, "kv-test").In("foo"))

		conn, err := cm.Get(ctx, "some-type", "test")
		require.NoError(t, err)
		require.Equal(t, "some-type", conn.Type)
		require.Equal(t, "test", conn.Name)
		require.Equal(t, attrs, conn.Attributes)

		_, err = cm.Get(ctx, "some-other-type", "test")
		require.Equal(t, model.ErrNotFound, err)

		conns, err := cm.List(ctx)
		require.NoError(t, err)
		require.Len(t, conns, 1)
		require.Contains(t, conns, &model.Connection{Type: "some-type", Name: "test", Attributes: attrs})
	})
}
