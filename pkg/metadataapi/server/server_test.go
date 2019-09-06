package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/horsehead/encoding/transfer"
	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type mockTaskConfig struct {
	Name      string
	Namespace string
	Labels    map[string]string
	PodIP     string
	SpecData  map[string]interface{}
}

func mockTask(t *testing.T, cfg mockTaskConfig) []runtime.Object {
	specData, err := json.Marshal(cfg.SpecData)
	require.NoError(t, err)

	return []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.Name,
				Namespace: cfg.Namespace,
				Labels:    cfg.Labels,
			},
			Status: corev1.PodStatus{
				PodIP: cfg.PodIP,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cfg.Name,
				Namespace: cfg.Namespace,
			},
			Data: map[string]string{
				"spec.json": string(specData),
			},
		},
	}
}

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

type mockMetadataManager struct {
	taskName string
}

func (mm mockMetadataManager) Get(context.Context) (*task.Metadata, errors.Error) {
	return &task.Metadata{Name: mm.taskName}, nil
}

type mockManagerFactoryConfig struct {
	taskName     string
	secretData   map[string]string
	namespace    string
	k8sResources []runtime.Object
}

type mockManagerFactory struct {
	sm op.SecretsManager
	om op.OutputsManager
	mm op.MetadataManager
	km op.KubernetesManager
}

func (m mockManagerFactory) SecretsManager() op.SecretsManager {
	return m.sm
}

func (m mockManagerFactory) OutputsManager() op.OutputsManager {
	return m.om
}

func (m mockManagerFactory) MetadataManager() op.MetadataManager {
	return nil
}

func (m mockManagerFactory) KubernetesManager() op.KubernetesManager {
	return m.km
}

func newMockManagerFactory(t *testing.T, cfg mockManagerFactoryConfig) mockManagerFactory {
	km := op.NewDefaultKubernetesManager(cfg.namespace, fake.NewSimpleClientset(cfg.k8sResources...))
	om := configmap.New(km.Client(), cfg.namespace)

	return mockManagerFactory{
		sm: op.NewEncodingSecretManager(mockSecretsManager{
			data: cfg.secretData,
		}),
		om: om,
		km: km,
	}
}

func withTestAPIServer(managers op.ManagerFactory, fn func(*httptest.Server)) {
	srv := New(&config.MetadataServerConfig{
		Logger:    logging.Builder().At("server-test").Build(),
		Namespace: managers.KubernetesManager().Namespace(),
	}, managers)

	ts := httptest.NewServer(srv)

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

func TestServerOutputsHandler(t *testing.T) {
	managers := newMockManagerFactory(t, mockManagerFactoryConfig{
		k8sResources: mockTask(t, mockTaskConfig{
			Name:      "test-task",
			Namespace: "test-task",
			Labels: map[string]string{
				"task-name": "test-task",
			},
			PodIP: "10.3.3.3",
		}),
	})

	withTestAPIServer(managers, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/foo", strings.NewReader("bar"))
		require.NoError(t, err)

		req.Header.Set("X-Forwarded-For", "10.3.3.3")

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		resp, err = client.Get(ts.URL + "/outputs/test-task/foo")
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
	const (
		namespace    = "workflow-run-ns"
		currentTask  = "current-task"
		previousTask = "previous-task"
	)

	resources := []runtime.Object{}

	resources = append(resources, mockTask(t, mockTaskConfig{
		Name:      previousTask,
		Namespace: namespace,
		Labels: map[string]string{
			"task-name": previousTask,
		},
		PodIP: "10.3.3.3",
	})...)

	resources = append(resources, mockTask(t, mockTaskConfig{
		Name:      currentTask,
		Namespace: namespace,
		Labels: map[string]string{
			"task-name": currentTask,
		},
		PodIP: "10.3.3.4",
		SpecData: map[string]interface{}{
			"super-secret": map[string]string{"$type": "Secret", "name": "test-secret-key"},
			"super-output": map[string]string{"$type": "Output", "name": "test-output-key", "taskName": previousTask},
			"super-normal": "test-normal-value",
		},
	})...)

	managers := newMockManagerFactory(t, mockManagerFactoryConfig{
		taskName: currentTask,
		secretData: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		namespace:    namespace,
		k8sResources: resources,
	})

	withTestAPIServer(managers, func(ts *httptest.Server) {
		client := ts.Client()

		req, err := http.NewRequest(http.MethodPut, ts.URL+"/outputs/test-output-key", strings.NewReader("test-output-value"))
		require.NoError(t, err)

		req.Header.Set("X-Forwarded-For", "10.3.3.3")

		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		resp, err = client.Get(ts.URL + "/specs/" + currentTask)
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
