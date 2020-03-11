package configmap_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/outputs"
	"github.com/stretchr/testify/require"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/testutil"
)

func TestConfigMapOutputs(t *testing.T) {
	t.Parallel()

	taskConfig := testutil.MockTaskConfig{
		Run:       uuid.New().String(),
		Name:      "test-task",
		Namespace: "test-task",
		PodIP:     "10.3.3.3",
	}

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    "test-task",
		K8sResources: testutil.MockTask(t, taskConfig),
	})
	logger := logging.Builder().At("outputs-client-test").Build()
	srv := server.New(&config.MetadataServerConfig{Logger: logger}, managers)
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
			key:         "foo",
			value:       "bar",
			taskName:    taskConfig.Name,
		},
		{
			description: "missing key raises an error",
			key:         "",
			value:       "bar",
			taskName:    taskConfig.Name,
			setErr:      outputs.ErrOutputsClientKeyEmpty,
		},
		{
			description: "missing value raises an error",
			key:         "foo",
			value:       "",
			taskName:    taskConfig.Name,
			setErr:      outputs.ErrOutputsClientValueEmpty,
		},
		{
			description: "missing taskName raises an error",
			key:         "foo",
			value:       "value",
			taskName:    "",
			getErr:      outputs.ErrOutputsClientTaskNameEmpty,
		},
	}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		for _, c := range cases {
			t.Run(c.description, func(t *testing.T) {
				apiEndpoint, err := url.Parse(ts.URL + "/outputs")
				require.NoError(t, err)

				client := outputs.NewDefaultOutputsClient(apiEndpoint)
				ctx := context.Background()

				err = client.SetOutput(ctx, c.key, c.value)
				if c.setErr != nil {
					require.Error(t, err)
					require.Equal(t, c.setErr, err)

					return
				}

				require.NoError(t, err)

				value, err := client.GetOutput(ctx, c.taskName, c.key)
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
