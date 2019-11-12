package configmap

import (
	"context"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type StateManager struct {
	namespace  string
	kubeclient kubernetes.Interface
}

func (sm StateManager) Get(ctx context.Context, stepName, key string) (*outputs.Output, errors.Error) {
	cm, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Get(stepName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewOutputsTaskNotFound(stepName).WithCause(err)
		}

		return nil, errors.NewOutputsGetError().WithCause(err)
	}

	val, ok := cm.Data[key]
	if !ok {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		Key:      key,
		Value:    val,
		TaskName: stepName,
	}, nil
}

func New(kc kubernetes.Interface, namespace string) *StateManager {
	return &StateManager{
		kubeclient: kc,
		namespace:  namespace,
	}
}
