package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/testutil"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
)

func TestStateManager(t *testing.T) {
	t.Parallel()

	taskConfig := testutil.MockTaskConfig{
		Name:      "test-task",
		Namespace: "test-task",
		PodIP:     "10.3.3.3",
	}

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    taskConfig.Namespace,
		K8sResources: testutil.MockTask(t, taskConfig),
	})
	logger := logging.Builder().At("state-client-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)
	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddress(taskConfig.PodIP)}

	cases := []struct {
		description string
		key         string
		value       string
		taskName    string
		setErr      error
		getErr      error
	}{
		{
			description: "can set a simple pair",
			key:         "test-key",
			value:       "test-value",
			taskName:    taskConfig.Name,
		},
		{
			description: "missing key raises an error",
			key:         "",
			value:       "test-value",
			taskName:    taskConfig.Name,
			getErr:      state.ErrStateClientKeyEmpty,
		},
	}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		for _, c := range cases {
			t.Run(c.description, func(t *testing.T) {
				apiEndpoint, err := url.Parse(ts.URL + "/state")
				require.NoError(t, err)

				client := state.NewDefaultStateClient(apiEndpoint)
				ctx := context.Background()

				data := make(map[string]interface{})
				data[c.key] = c.value
				raw, err := json.Marshal(data)
				require.NoError(t, err)

				buf := &bytes.Buffer{}
				buf.Write(raw)

				err = client.SetState(ctx, buf.String())
				if c.setErr != nil {
					require.Error(t, err)
					require.Equal(t, c.setErr, err)

					return
				}

				require.NoError(t, err)

				value, err := client.GetState(ctx, c.key)
				if c.getErr != nil {
					require.Error(t, err)
					require.Equal(t, c.getErr, err)

					return
				}

				require.NoError(t, err)
				require.Equal(t, c.value, value)
			})
		}
	})
}
