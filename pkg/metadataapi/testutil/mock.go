package testutil

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	smemory "github.com/puppetlabs/nebula-tasks/pkg/secrets/memory"
	stconfigmap "github.com/puppetlabs/nebula-tasks/pkg/state/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

type MockTaskConfig struct {
	ID        string
	Name      string
	Namespace string
	PodIP     string
	SpecData  map[string]interface{}
}

func MockTask(t *testing.T, cfg MockTaskConfig) []runtime.Object {
	specData, err := json.Marshal(cfg.SpecData)
	require.NoError(t, err)

	taskHash := sha1.Sum([]byte(cfg.Name))

	labels := map[string]string{
		"nebula.puppet.com/task.hash": hex.EncodeToString(taskHash[:]),
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

type MockManagerFactoryConfig struct {
	SecretData   map[string]string
	Namespace    string
	K8sResources []runtime.Object
}

type MockManagerFactory struct {
	sm  op.SecretsManager
	om  op.OutputsManager
	stm op.StateManager
	mm  op.MetadataManager
	spm op.SpecsManager
}

func (m MockManagerFactory) SecretsManager() op.SecretsManager {
	return m.sm
}

func (m MockManagerFactory) OutputsManager() op.OutputsManager {
	return m.om
}

func (m MockManagerFactory) StateManager() op.StateManager {
	return m.stm
}

func (m MockManagerFactory) MetadataManager() op.MetadataManager {
	return m.mm
}

func (m MockManagerFactory) SpecsManager() op.SpecsManager {
	return m.spm
}

func NewMockManagerFactory(t *testing.T, cfg MockManagerFactoryConfig) MockManagerFactory {
	kc := fake.NewSimpleClientset(cfg.K8sResources...)
	kc.PrependReactor("create", "*", setObjectUID)

	om := configmap.New(kc, cfg.Namespace)
	stm := stconfigmap.New(kc, cfg.Namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.Namespace)
	sm := smemory.New(cfg.SecretData)
	spm := task.NewKubernetesSpecManager(kc, cfg.Namespace)

	return MockManagerFactory{
		sm:  op.NewEncodingSecretManager(sm),
		om:  om,
		stm: op.NewEncodeDecodingStateManager(stm),
		mm:  mm,
		spm: spm,
	}
}

func WithRemoteAddress(ip string) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, port, _ := net.SplitHostPort(r.RemoteAddr)
			r.RemoteAddr = strings.Join([]string{host, port}, ":")

			next.ServeHTTP(w, r)
		})
	}
}

func WithTestMetadataAPIServer(handler http.Handler, mw []middleware.MiddlewareFunc, fn func(*httptest.Server)) {
	for _, m := range mw {
		handler = m(handler)
	}

	ts := httptest.NewServer(handler)

	defer ts.Close()

	fn(ts)
}

func setObjectUID(action kubetesting.Action) (bool, runtime.Object, error) {
	switch action := action.(type) {
	case kubetesting.CreateActionImpl:
		objMeta, err := meta.Accessor(action.GetObject())
		if err != nil {
			return false, nil, err
		}

		obj := action.GetObject()
		objMeta.SetUID(types.UID(uuid.New().String()))

		return false, obj, nil
	default:
		return false, nil, fmt.Errorf("no reaction implemented for %s", action)
	}
}
