package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/horsehead/encoding/transfer"
	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/stretchr/testify/require"
)

type mockSecretsManager struct {
	data map[string]string
}

func (sm mockSecretsManager) Get(ctx context.Context, key string) (*secrets.Secret, errors.Error) {
	val, ok := sm.data[key]
	if !ok {
		return nil, errors.NewSecretsKeyNotFound(key)
	}

	sec := &secrets.Secret{
		Key:   key,
		Value: val,
	}

	return sec, nil
}

type mockManagerFactory struct {
	sm op.SecretsManager
}

func (m mockManagerFactory) SecretsManager() op.SecretsManager {
	return m.sm
}

func newMockManagerFactory(secretData map[string]string) mockManagerFactory {
	return mockManagerFactory{
		sm: op.NewEncodingSecretManager(mockSecretsManager{
			data: secretData,
		}),
	}
}

func withTestAPIServer(managers op.ManagerFactory, fn func(*httptest.Server)) {
	srv := New(&config.MetadataServerConfig{
		Logger:    logging.Builder().At("server-test").Build(),
		Namespace: "test",
	}, managers)

	ts := httptest.NewServer(srv)

	defer ts.Close()

	fn(ts)
}

func TestServer(t *testing.T) {
	encodedBar, err := transfer.EncodeForTransfer([]byte("bar"))
	require.NoError(t, err)

	managers := newMockManagerFactory(map[string]string{
		"foo": encodedBar,
	})

	withTestAPIServer(managers, func(ts *httptest.Server) {
		client := ts.Client()

		resp, err := client.Get(ts.URL + "/secrets/foo")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		var sec secrets.Secret
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&sec))

		require.Equal(t, "foo", sec.Key)
		require.Equal(t, "bar", sec.Value)
	})
}
