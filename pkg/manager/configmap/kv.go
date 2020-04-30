package configmap

import (
	"context"
	"encoding/json"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type KVConfigMap struct {
	cm ConfigMap
}

func (kcm *KVConfigMap) Get(ctx context.Context, key string) (interface{}, error) {
	cm, err := kcm.cm.Get(ctx)
	if errors.IsNotFound(err) {
		return nil, model.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	encoded, found := cm.Data[key]
	if !found {
		return nil, model.ErrNotFound
	}

	var value transfer.JSONInterface
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return nil, err
	}

	return value.Data, nil
}

func (kcm *KVConfigMap) Set(ctx context.Context, key string, value interface{}) error {
	encoded, err := json.Marshal(transfer.JSONInterface{Data: value})
	if err != nil {
		return err
	}

	if _, err := MutateConfigMap(ctx, kcm.cm, func(cm *corev1.ConfigMap) {
		cm.Data[key] = string(encoded)
	}); err != nil {
		return err
	}

	return nil
}

func NewKVConfigMap(backend ConfigMap) *KVConfigMap {
	kcm := &KVConfigMap{
		cm: backend,
	}

	return kcm
}
