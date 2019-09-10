package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/horsehead/encoding/transfer"
	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	smemory "github.com/puppetlabs/nebula-tasks/pkg/secrets/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type mockTaskConfig struct {
	ID        string
	Name      string
	Namespace string
	PodIP     string
	SpecData  map[string]interface{}
}

func mockTask(t *testing.T, cfg mockTaskConfig) []runtime.Object {
	specData, err := json.Marshal(cfg.SpecData)
	require.NoError(t, err)

	labels := map[string]string{
		"task-id":   cfg.ID,
		"task-name": cfg.Name,
	}

	return []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.Name,
				Namespace: cfg.Namespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				PodIP: cfg.PodIP,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.ID,
				Namespace: cfg.Namespace,
				Labels:    labels,
			},
			Data: map[string]string{
				"spec.json": string(specData),
			},
		},
	}
}

type mockManagerFactoryConfig struct {
	secretData   map[string]string
	namespace    string
	k8sResources []runtime.Object
}

type mockManagerFactory struct {
	sm  op.SecretsManager
	om  op.OutputsManager
	mm  op.MetadataManager
	spm op.SpecsManager
}

func (m mockManagerFactory) SecretsManager() op.SecretsManager {
	return m.sm
}

func (m mockManagerFactory) OutputsManager() op.OutputsManager {
	return m.om
}

func (m mockManagerFactory) MetadataManager() op.MetadataManager {
	return m.mm
}

func (m mockManagerFactory) SpecsManager() op.SpecsManager {
	return m.spm
}

func newMockManagerFactory(t *testing.T, cfg mockManagerFactoryConfig) mockManagerFactory {
	kc := fake.NewSimpleClientset(cfg.k8sResources...)
	om := configmap.New(kc, cfg.namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.namespace)
	sm := smemory.New(cfg.secretData)
	spm := task.NewKubernetesSpecManager(kc, cfg.namespace)

	return mockManagerFactory{
		sm:  op.NewEncodingSecretManager(sm),
		om:  op.NewEncodeDecodingOutputsManager(om),
		mm:  mm,
		spm: spm,
	}
}

func withRemoteAddress(ip string) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, port, _ := net.SplitHostPort(r.RemoteAddr)
			r.RemoteAddr = strings.Join([]string{host, port}, ":")

			next.ServeHTTP(w, r)
		})
	}
}

func withTestAPIServer(managers op.ManagerFactory, mw []middleware.MiddlewareFunc, fn func(*httptest.Server)) {
	logger := logging.Builder().At("server-test").Build()

	srv := New(&config.MetadataServerConfig{Logger: logger}, managers)

	var handler http.Handler

	handler = srv
	for _, m := range mw {
		handler = m(handler)
	}

	ts := httptest.NewServer(handler)

	defer ts.Close()

	fn(ts)
}

func TestServerSecretsHandler(t *testing.T) {
	encodedBar, err := transfer.EncodeForTransfer([]byte("bar"))
	require.NoError(t, err)

	managers := newMockManagerFactory(t, mockManagerFactoryConfig{
		secretData: map[string]string{
			"foo": encodedBar,
		},
	})

	withTestAPIServer(managers, []middleware.MiddlewareFunc{}, func(ts *httptest.Server) {
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

func TestServerOutputsHandler(t *testing.T) {
	taskConfig := mockTaskConfig{
		ID:        uuid.New().String(),
		Name:      "test-task",
		Namespace: "test-task",
		PodIP:     "10.3.3.3",
	}

	managers := newMockManagerFactory(t, mockManagerFactoryConfig{
		k8sResources: mockTask(t, taskConfig),
	})

	mw := []middleware.MiddlewareFunc{withRemoteAddress(taskConfig.PodIP)}

	withTestAPIServer(managers, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/foo", strings.NewReader("bar"))
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
		require.Equal(t, "bar", out.Value)
		require.Equal(t, "test-task", out.TaskName)
	})
}

func TestServerSpecsHandler(t *testing.T) {
	const namespace = "workflow-run-ns"

	var (
		previousTask = mockTaskConfig{
			ID:        uuid.New().String(),
			Name:      "previous-task",
			Namespace: namespace,
			PodIP:     "10.3.3.4",
		}
		currentTask = mockTaskConfig{
			ID:        uuid.New().String(),
			Name:      "current-task",
			Namespace: namespace,
			PodIP:     "10.3.3.3",
			SpecData: map[string]interface{}{
				"super-secret": map[string]string{"$type": "Secret", "name": "test-secret-key"},
				"super-output": map[string]string{"$type": "Output", "name": "test-output-key", "taskName": previousTask.Name},
				"super-normal": "test-normal-value",
			},
		}
	)

	resources := []runtime.Object{}
	resources = append(resources, mockTask(t, previousTask)...)
	resources = append(resources, mockTask(t, currentTask)...)

	managers := newMockManagerFactory(t, mockManagerFactoryConfig{
		secretData: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		namespace:    namespace,
		k8sResources: resources,
	})

	mw := []middleware.MiddlewareFunc{withRemoteAddress(previousTask.PodIP)}

	withTestAPIServer(managers, mw, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/test-output-key", strings.NewReader("test-output-value"))
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		resp, err = client.Get(ts.URL + "/specs/" + currentTask.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		defer resp.Body.Close()

		// the test spec, after interpolation from the api, should be a more flat map[string]string,
		// so we will try to unmarshal the response into something like that to see if
		// that's what we got.
		spec := make(map[string]string)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&spec))
		require.Equal(t, "test-secret-value", spec["super-secret"])
		require.Equal(t, "test-output-value", spec["super-output"])
		require.Equal(t, "test-normal-value", spec["super-normal"])
	})
}
