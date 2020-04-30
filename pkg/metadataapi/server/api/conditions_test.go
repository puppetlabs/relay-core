package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	sdktestutil "github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/testutil"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/puppetlabs/nebula-tasks/pkg/conditionals"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/testutil"
)

func TestConditionalsHandlerSuccess(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		run          = uuid.New().String()
		previousTask = testutil.MockTaskConfig{
			Run:       run,
			Name:      "previous-task",
			Namespace: namespace,
			PodIP:     "10.3.3.4",
		}
		task = testutil.MockTaskConfig{
			Run:       run,
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			When: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONOutput(previousTask.Name, "output1"),
					"foobar",
				}),
				sdktestutil.JSONInvocation("notEquals", []interface{}{
					sdktestutil.JSONOutput(previousTask.Name, "output1"),
					"barfoo",
				}),
			},
			SpecData: map[string]interface{}{
				"super-normal": "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, previousTask)...)
	resources = append(resources, testutil.MockTask(t, task)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    namespace,
		K8sResources: resources,
	})

	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddressFromHeader("Nebula-Unit-Test-Address")}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/output1", strings.NewReader("foobar"))
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", previousTask.PodIP)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		req, err = http.NewRequest(http.MethodGet, ts.URL+"/conditions", nil)
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", task.PodIP)

		resp, err = client.Do(req)

		defer resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		env := conditionals.ResponseEnvelope{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
		require.Equal(t, true, env.Success)
	})
}

func TestConditionalsHandlerFailure(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		run          = uuid.New().String()
		previousTask = testutil.MockTaskConfig{
			Run:       run,
			Name:      "previous-task",
			Namespace: namespace,
			PodIP:     "10.3.3.4",
		}
		task = testutil.MockTaskConfig{
			Run:       run,
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			When: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONOutput(previousTask.Name, "output1"),
					"foobar",
				}),
				sdktestutil.JSONInvocation("notEquals", []interface{}{
					sdktestutil.JSONOutput(previousTask.Name, "output1"),
					"foobar",
				}),
			},
			SpecData: map[string]interface{}{
				"super-normal": "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, previousTask)...)
	resources = append(resources, testutil.MockTask(t, task)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    namespace,
		K8sResources: resources,
	})

	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddressFromHeader("Nebula-Unit-Test-Address")}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/output1", strings.NewReader("foobar"))
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", previousTask.PodIP)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		req, err = http.NewRequest(http.MethodGet, ts.URL+"/conditions", nil)
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", task.PodIP)

		resp, err = client.Do(req)

		defer resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		env := conditionals.ResponseEnvelope{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
		require.Equal(t, false, env.Success)
	})
}

func TestConditionalsHandlerUnsupportedExpressions(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		task = testutil.MockTaskConfig{
			Run:       uuid.New().String(),
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			When: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONSecret("secret1"),
					"foobar",
				}),
				sdktestutil.JSONInvocation("notEquals", []interface{}{
					sdktestutil.JSONSecret("secret2"),
					"foobar",
				}),
			},
			SpecData: map[string]interface{}{
				"super-normal": "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, task)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    namespace,
		K8sResources: resources,
	})

	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddressFromHeader("Nebula-Unit-Test-Address")}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/conditions", nil)
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", task.PodIP)

		resp, err := client.Do(req)

		defer resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var env utilapi.ErrorEnvelope

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
		require.Equal(t, `unsupported_conditional_expressions: One or more expressions are not supported:
* !Secret secret1
* !Secret secret2`, env.Error.AsError().Error())
	})
}

func TestConditionalsHandlerUnresolvedExpressions(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		task = testutil.MockTaskConfig{
			Run:       uuid.New().String(),
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			When: sdktestutil.JSONInvocation("equals", []interface{}{
				sdktestutil.JSONParameter("param1"),
				"foobar",
			}),
			SpecData: map[string]interface{}{
				"super-normal": "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, testutil.MockTask(t, task)...)

	managers := testutil.NewMockManagerFactory(t, testutil.MockManagerFactoryConfig{
		Namespace:    namespace,
		K8sResources: resources,
	})

	logger := logging.Builder().At("server-test").Build()
	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	mw := []middleware.MiddlewareFunc{testutil.WithRemoteAddressFromHeader("Nebula-Unit-Test-Address")}

	testutil.WithTestMetadataAPIServer(srv, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/conditions", nil)
		require.NoError(t, err)
		req.Header.Set("Nebula-Unit-Test-Address", task.PodIP)

		resp, err := client.Do(req)

		defer resp.Body.Close()

		require.NoError(t, err)
		require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var env utilapi.ErrorEnvelope

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
		require.Equal(t, `unresolved_conditional_expressions: One or more expressions were unresolvable:
* resolve: parameter "param1" could not be found`, env.Error.AsError().Error())
	})
}
