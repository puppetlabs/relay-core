package configmap

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigMap interface {
	Get(ctx context.Context) (*corev1.ConfigMap, error)
	CreateOrUpdate(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error)
}

type ClientConfigMap struct {
	client          kubernetes.Interface
	namespace, name string
}

var _ ConfigMap = &ClientConfigMap{}

func (ccm *ClientConfigMap) Get(ctx context.Context) (*corev1.ConfigMap, error) {
	return ccm.client.CoreV1().ConfigMaps(ccm.namespace).Get(ccm.name, metav1.GetOptions{})
}

func (ccm *ClientConfigMap) CreateOrUpdate(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cm.SetNamespace(ccm.namespace)
	cm.SetName(ccm.name)

	if len(cm.GetUID()) == 0 {
		return ccm.client.CoreV1().ConfigMaps(ccm.namespace).Create(cm)
	}

	return ccm.client.CoreV1().ConfigMaps(ccm.namespace).Update(cm)
}

func NewClientConfigMap(client kubernetes.Interface, namespace, name string) *ClientConfigMap {
	return &ClientConfigMap{
		client:    client,
		namespace: namespace,
		name:      name,
	}
}

type ControllerRuntimeConfigMap struct {
	client client.Client
	key    client.ObjectKey
}

var _ ConfigMap = &ControllerRuntimeConfigMap{}

func (crcm *ControllerRuntimeConfigMap) Get(ctx context.Context) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}

	if err := crcm.client.Get(ctx, crcm.key, cm); err != nil {
		return nil, err
	}

	return cm, nil
}

func (crcm *ControllerRuntimeConfigMap) CreateOrUpdate(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cm.SetNamespace(crcm.key.Namespace)
	cm.SetName(crcm.key.Name)

	if len(cm.GetUID()) == 0 {
		if err := crcm.client.Create(ctx, cm); err != nil {
			return nil, err
		}
	} else {
		if err := crcm.client.Update(ctx, cm); err != nil {
			return nil, err
		}
	}

	return cm, nil
}

func NewControllerRuntimeConfigMap(cl client.Client, key client.ObjectKey) *ControllerRuntimeConfigMap {
	return &ControllerRuntimeConfigMap{
		client: cl,
		key:    key,
	}
}

func MutateConfigMap(ctx context.Context, cm ConfigMap, fn func(cm *corev1.ConfigMap)) (*corev1.ConfigMap, error) {
	for {
		obj, err := cm.Get(ctx)
		if errors.IsNotFound(err) {
			obj = &corev1.ConfigMap{
				Data: make(map[string]string),
			}
		} else if err != nil {
			return nil, err
		}

		fn(obj)

		obj, err = cm.CreateOrUpdate(ctx, obj)
		if errors.IsConflict(err) || errors.IsNotFound(err) {
			// Updated/deleted from under us. Try again.
			continue
		} else if err != nil {
			return nil, err
		}

		return obj, nil
	}
}
