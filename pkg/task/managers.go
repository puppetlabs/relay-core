package task

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
)

// PreconfiguredMetadataManager is a task metadata manager that can be
// pre-populated for use in development and testing.
type PreconfiguredMetadataManager struct {
	tasks map[string]*Metadata
}

func (m PreconfiguredMetadataManager) GetByIP(ctx context.Context, ip string) (*Metadata, errors.Error) {
	if m.tasks == nil {
		return nil, errors.NewTaskNotFoundForIP(ip)
	}

	if task, ok := m.tasks[ip]; ok {
		return task, nil
	}

	return nil, errors.NewTaskNotFoundForIP(ip)
}

func NewPreconfiguredMetadataManager(tasks map[string]*Metadata) *PreconfiguredMetadataManager {
	return &PreconfiguredMetadataManager{tasks: tasks}
}

// KubernetesMetadataManager provides metadata about a task by introspecting
// the Kubernetes pod it runs in using regular resource apis and kube
// clients.
type KubernetesMetadataManager struct {
	kubeclient kubernetes.Interface
	namespace  string
}

func (mm *KubernetesMetadataManager) GetByIP(ctx context.Context, ip string) (*Metadata, errors.Error) {
	listOpts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("status.podIP=%s", ip),
	}

	pods, err := mm.kubeclient.CoreV1().Pods(mm.namespace).List(listOpts)
	if err != nil {
		return nil, errors.NewKubernetesPodLookupError().WithCause(err)
	}

	if len(pods.Items) < 1 {
		return nil, errors.NewTaskNotFoundForIP(ip)
	}

	// TODO fine tune this: this should theoretically never return more than 1 pod (if it does,
	// then our network fabric has some serious issues), but we should figure out how to handle
	// this scenario.
	pod := pods.Items[0]

	taskHashHex := pod.GetLabels()["nebula.puppet.com/task.hash"]
	run := pod.GetLabels()["nebula.puppet.com/run"]
	taskHash, err := hex.DecodeString(taskHashHex)
	if err != nil {
		return nil, errors.NewTaskInvalidHashError().WithCause(err).Bug()
	} else if len(taskHash) != sha1.Size {
		return nil, errors.NewTaskInvalidHashError().Bug()
	}

	md := &Metadata{Run: run}
	copy(md.Hash[:], taskHash)

	return md, nil
}

func NewKubernetesMetadataManager(kubeclient kubernetes.Interface, namespace string) *KubernetesMetadataManager {
	return &KubernetesMetadataManager{
		kubeclient: kubeclient,
		namespace:  namespace,
	}
}

type PreconfiguredSpecManager struct {
	specs map[string]string
}

func (sm PreconfiguredSpecManager) Get(ctx context.Context, metadata *Metadata) (string, errors.Error) {
	taskHashKey := metadata.Hash.HexEncoding()

	if sm.specs == nil {
		return "", errors.NewTaskSpecNotFoundForID(taskHashKey)
	}

	if _, ok := sm.specs[taskHashKey]; !ok {
		return "", errors.NewTaskSpecNotFoundForID(taskHashKey)
	}

	return sm.specs[taskHashKey], nil
}

func NewPreconfiguredSpecManager(specs map[string]string) *PreconfiguredSpecManager {
	return &PreconfiguredSpecManager{specs: specs}
}

type KubernetesSpecManager struct {
	kubeclient kubernetes.Interface
	namespace  string
}

func (sm KubernetesSpecManager) Get(ctx context.Context, metadata *Metadata) (string, errors.Error) {
	taskHashKey := metadata.Hash.HexEncoding()

	configMap, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Get(taskHashKey, metav1.GetOptions{})
	if nil != err {
		if kerrors.IsNotFound(err) {
			return "", errors.NewTaskSpecNotFoundForID(taskHashKey).WithCause(err)
		}

		return "", errors.NewTaskSpecLookupError().WithCause(err)
	}

	if _, ok := configMap.Data["spec.json"]; !ok {
		return "", errors.NewTaskSpecNotFoundForID(taskHashKey)
	}

	return configMap.Data["spec.json"], nil
}

func NewKubernetesSpecManager(kubeclient kubernetes.Interface, namespace string) *KubernetesSpecManager {
	return &KubernetesSpecManager{
		kubeclient: kubeclient,
		namespace:  namespace,
	}
}
