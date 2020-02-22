package configmap

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OutputsManager is an abstraction on top of the K8s api's for storing
// data as key/value pairs in configmap resources.
type OutputsManager struct {
	// namespace is the kubernetes namespace to scope queries in.
	namespace string
	// kubeclient is the kubernetes clientset used to access configmap resources
	kubeclient kubernetes.Interface
}

func (om OutputsManager) Get(ctx context.Context, taskName, key string) (*outputs.Output, errors.Error) {
	taskHash := sha1.Sum([]byte(taskName))
	name := fmt.Sprintf("task-%x-outputs", taskHash)

	cm, err := om.kubeclient.CoreV1().ConfigMaps(om.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewOutputsTaskNotFound(taskName).WithCause(err)
		}

		return nil, errors.NewOutputsGetError().WithCause(err)
	}

	s, ok := cm.Data[key]
	if !ok {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	var val transfer.JSONInterface
	if err := json.Unmarshal([]byte(s), &val); err != nil {
		return nil, errors.NewOutputsValueDecodingError().WithCause(err).Bug()
	}

	return &outputs.Output{
		Key:      key,
		Value:    val,
		TaskName: taskName,
	}, nil
}

func (om OutputsManager) Put(ctx context.Context, taskHash [sha1.Size]byte, key string, value transfer.JSONInterface) errors.Error {
	name := fmt.Sprintf("task-%x-outputs", taskHash)

	cm, err := om.kubeclient.CoreV1().ConfigMaps(om.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: om.namespace,
				},
				Data: map[string]string{},
			}
		} else {
			return errors.NewOutputsPutError().WithCause(err)
		}
	}

	b, err := json.Marshal(value)
	if err != nil {
		return errors.NewOutputsValueEncodingError().WithCause(err).Bug()
	}

	cm.Data[key] = string(b)

	if err := om.createOrUpdateConfigMap(cm); err != nil {
		return errors.NewOutputsPutError().WithCause(err)
	}

	return nil
}

func (om OutputsManager) createOrUpdateConfigMap(cm *corev1.ConfigMap) error {
	if string(cm.GetUID()) == "" {
		_, err := om.kubeclient.CoreV1().ConfigMaps(om.namespace).Create(cm)

		return err
	}

	_, err := om.kubeclient.CoreV1().ConfigMaps(om.namespace).Update(cm)

	return err
}

func New(kc kubernetes.Interface, namespace string) *OutputsManager {
	return &OutputsManager{
		kubeclient: kc,
		namespace:  namespace,
	}
}
