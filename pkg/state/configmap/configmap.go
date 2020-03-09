package configmap

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type StateManager struct {
	namespace  string
	kubeclient kubernetes.Interface
}

func (sm StateManager) Get(ctx context.Context, metadata *task.Metadata, key string) (*state.State, errors.Error) {
	taskHashKey := metadata.Hash.HexEncoding()

	cm, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Get(taskHashKey, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewStateTaskNotFound(taskHashKey).WithCause(err)
		}

		return nil, errors.NewStateGetError().WithCause(err)
	}

	if _, ok := cm.Data["state"]; !ok {
		return nil, errors.NewStateNotFoundForID(taskHashKey)
	}

	st := cm.Data["state"]

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(st), &data); err != nil {
		return nil, errors.NewStateValueDecodingError().WithCause(err)
	}

	val, ok := data[key]
	if !ok {
		return nil, errors.NewStateKeyNotFound(key)
	}

	return &state.State{
		Key:   key,
		Value: val.(string),
	}, nil
}

func (sm StateManager) Set(ctx context.Context, metadata *task.Metadata, value io.Reader) errors.Error {
	taskHashKey := metadata.Hash.HexEncoding()

	cm, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Get(taskHashKey, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskHashKey,
					Namespace: sm.namespace,
				},
				Data: map[string]string{},
			}
		} else {
			return errors.NewStatePutError().WithCause(err)
		}
	}

	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(value); err != nil {
		return errors.NewStateValueReadError().WithCause(err)
	}

	cm.Data["state"] = buf.String()

	if err := sm.createOrUpdateConfigMap(cm); err != nil {
		return errors.NewStatePutError().WithCause(err)
	}

	return nil
}

func (sm StateManager) createOrUpdateConfigMap(cm *corev1.ConfigMap) error {
	if string(cm.GetUID()) == "" {
		_, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Create(cm)

		return err
	}

	_, err := sm.kubeclient.CoreV1().ConfigMaps(sm.namespace).Update(cm)

	return err
}

func New(kc kubernetes.Interface, namespace string) *StateManager {
	return &StateManager{
		kubeclient: kc,
		namespace:  namespace,
	}
}
