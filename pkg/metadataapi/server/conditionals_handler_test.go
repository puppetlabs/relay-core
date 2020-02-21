package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/testutil"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestConditionalsHandler(t *testing.T) {
	const namespace = "workflow-run-ns"

	var task = testutil.MockTaskConfig{
		ID:        uuid.New().String(),
		Name:      "current-task",
		Namespace: namespace,
		PodIP:     "10.3.3.3",
		When: map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"$fn.equals": []interface{}{
						map[string]string{"$type": "Parameter", "name": "param1"},
						"foobar",
					},
				},
			},
		},
		SpecData: map[string]interface{}{
			"super-normal": "test-normal-value",
		},
	}

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, task)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    namespace,
		K8sResources: resources,
	})

	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	testutil.WithTestMetadataAPIServer(srv, []middleware.MiddlewareFunc{}, func(ts *httptest.Server) {
		client := ts.Client()

		resp, err := client.Get(ts.URL + "/conditionals/" + task.ID)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
