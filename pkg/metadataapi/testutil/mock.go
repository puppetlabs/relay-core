package testutil

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	cconfigmap "github.com/puppetlabs/nebula-tasks/pkg/conditionals/configmap"
	cmemory "github.com/puppetlabs/nebula-tasks/pkg/connections/memory"
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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

type MockTaskConfig struct {
	Run       string
	Name      string
	Namespace string
	PodIP     string
	When      parse.Tree
	SpecData  map[string]interface{}
}

func (cfg *MockTaskConfig) TaskHash() task.Hash {
	thisTask := &task.Task{
		Run:  cfg.Run,
		Name: cfg.Name,
	}
	return thisTask.TaskHash()
}

func MockTask(t *testing.T, cfg MockTaskConfig) []runtime.Object {
	specData, err := json.Marshal(cfg.SpecData)
	require.NoError(t, err)

	conditionalsData, err := json.Marshal(cfg.When)
	require.NoError(t, err)

	taskHashKey := cfg.TaskHash().HexEncoding()

	labels := map[string]string{
		"nebula.puppet.com/task.hash": taskHashKey,
		"nebula.puppet.com/run":       cfg.Run,
	}

	return []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.New().String(),
				Namespace: cfg.Namespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				PodIP: cfg.PodIP,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskHashKey,
				Namespace: cfg.Namespace,
				Labels:    labels,
				UID:       types.UID(uuid.New().String()),
			},
			Data: map[string]string{
				"spec.json":    string(specData),
				"conditionals": string(conditionalsData),
			},
		},
	}
}

type MockManagerFactoryConfig struct {
	SecretData       map[string]string
	ConnectionData   map[string]map[string]interface{}
	ConditionalsData map[string]string
	Namespace        string
	K8sResources     []runtime.Object
}

type MockManagerFactory struct {
	sm  op.SecretsManager
	cm  op.ConnectionsManager
	om  op.OutputsManager
	stm op.StateManager
	mm  op.MetadataManager
	spm op.SpecsManager
	cdm op.ConditionalsManager
}

func (m MockManagerFactory) SecretsManager() op.SecretsManager {
	return m.sm
}

func (m MockManagerFactory) ConnectionsManager() op.ConnectionsManager {
	return m.cm
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

func (m MockManagerFactory) ConditionalsManager() op.ConditionalsManager {
	return m.cdm
}

func NewMockManagerFactory(t *testing.T, cfg MockManagerFactoryConfig) MockManagerFactory {
	kc := fake.NewSimpleClientset(cfg.K8sResources...)
	kc.PrependReactor("create", "*", setObjectUID)
	kc.PrependReactor("list", "pods", filterListPods(kc.Tracker()))

	om := configmap.New(kc, cfg.Namespace)
	stm := stconfigmap.New(kc, cfg.Namespace)
	mm := task.NewKubernetesMetadataManager(kc, cfg.Namespace)
	sm := smemory.New(cfg.SecretData)
	cm := cmemory.New(cfg.ConnectionData)
	spm := task.NewKubernetesSpecManager(kc, cfg.Namespace)
	cdm := cconfigmap.New(kc, cfg.Namespace)

	return MockManagerFactory{
		sm:  op.NewEncodingSecretManager(sm),
		cm:  cm,
		om:  om,
		stm: op.NewEncodeDecodingStateManager(stm),
		cdm: cdm,
		mm:  mm,
		spm: spm,
	}
}

func WithRemoteAddress(ip string) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, port, _ := net.SplitHostPort(r.RemoteAddr)
			r.RemoteAddr = strings.Join([]string{ip, port}, ":")

			next.ServeHTTP(w, r)
		})
	}
}

func WithRemoteAddressFromHeader(hdr string) middleware.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get(hdr)
			if ip != "" {
				WithRemoteAddress(ip)(next).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
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

func filterListPods(tracker kubetesting.ObjectTracker) kubetesting.ReactionFunc {
	delegate := kubetesting.ObjectReaction(tracker)

	return func(action kubetesting.Action) (bool, runtime.Object, error) {
		la := action.(kubetesting.ListAction)

		found, obj, err := delegate(action)
		if err != nil || !found {
			return found, obj, err
		}

		pods := obj.(*corev1.PodList)

		keep := 0
		for _, pod := range pods.Items {
			if !la.GetListRestrictions().Fields.Matches(fields.Set{"status.podIP": pod.Status.PodIP}) {
				continue
			}

			pods.Items[keep] = pod
			keep++
		}

		pods.Items = pods.Items[:keep]
		return true, pods, nil
	}
}
