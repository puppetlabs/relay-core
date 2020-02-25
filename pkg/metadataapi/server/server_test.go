package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/outputs"
	"github.com/puppetlabs/nebula-sdk/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/testutil"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestServerSecretsHandler(t *testing.T) {
	encodedBar, err := transfer.EncodeForTransfer([]byte("bar\x90"))
	require.NoError(t, err)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		SecretData: map[string]string{
			"foo": encodedBar,
		},
	})
	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	testutil.WithTestMetadataAPIServer(srv, []middleware.MiddlewareFunc{}, func(ts *httptest.Server) {
		client := ts.Client()

		// Get with valid secret.
		resp, err := client.Get(ts.URL + "/secrets/foo")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		var sec secrets.Secret
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&sec))

		require.Equal(t, "foo", sec.Key)

		v, err := sec.Value.Decode()
		require.NoError(t, err)
		require.Equal(t, "bar\x90", string(v))

		// Get with invalid secret.
		resp, err = client.Get(ts.URL + "/secrets/bar")
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)

		defer resp.Body.Close()
	})
}

func TestServerOutputsHandler(t *testing.T) {
	taskConfig := testutil.MockTaskConfig{
		ID:        uuid.New().String(),
		Name:      "test-task",
		Namespace: "test-task",
		PodIP:     "10.3.3.3",
	}

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    "test-task",
		K8sResources: testutil.MockTask(t, taskConfig),
	})
	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddress(taskConfig.PodIP)}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/foo", strings.NewReader("bar\x90"))
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		resp, err = client.Get(ts.URL + fmt.Sprintf("/outputs/%s/foo", taskConfig.Name))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		var out outputs.Output
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

		require.Equal(t, "foo", out.Key)
		require.Equal(t, "test-task", out.TaskName)

		require.NoError(t, err)
		require.Equal(t, "bar\x90", out.Value.Data)
	})
}

func TestServerSpecsHandler(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		previousTask = testutil.MockTaskConfig{
			ID:        uuid.New().String(),
			Name:      "previous-task",
			Namespace: namespace,
			PodIP:     "10.3.3.4",
		}
		currentTask = testutil.MockTaskConfig{
			ID:        uuid.New().String(),
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			SpecData: map[string]interface{}{
				"super-secret":      map[string]string{"$type": "Secret", "name": "test-secret-key"},
				"super-output":      map[string]string{"$type": "Output", "name": "test-output-key", "taskName": previousTask.Name},
				"structured-output": map[string]string{"$type": "Output", "name": "test-structured-output-key", "taskName": previousTask.Name},
				"super-normal":      "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, previousTask)...)
	resources = append(resources, testutil.MockTask(t, currentTask)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		SecretData: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		Namespace:    namespace,
		K8sResources: resources,
	})
	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddress(previousTask.PodIP)}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/test-output-key", strings.NewReader("test-output-value"))
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		req, err = http.NewRequest(http.MethodPut, ts.URL+"/outputs/test-structured-output-key", strings.NewReader(`{"a":"value","another":"thing"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resp, err = client.Get(ts.URL + "/specs/" + currentTask.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		spec := make(map[string]interface{})

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&spec))
		require.Equal(t, map[string]interface{}{
			"super-secret": "test-secret-value",
			"super-output": "test-output-value",
			"structured-output": map[string]interface{}{
				"a":       "value",
				"another": "thing",
			},
			"super-normal": "test-normal-value",
		}, spec)
	})
}
