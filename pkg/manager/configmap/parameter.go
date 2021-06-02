package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ParameterManager struct {
	kcm *KVConfigMap
}

var _ model.ParameterManager = &ParameterManager{}

func (m *ParameterManager) List(ctx context.Context) ([]*model.Parameter, error) {
	pm, err := m.kcm.List(ctx, parameterKey(""))
	if err != nil {
		return nil, err
	}

	var l []*model.Parameter

	for name, value := range pm {
		l = append(l, &model.Parameter{
			Name:  name,
			Value: value,
		})
	}

	return l, nil
}

func (m *ParameterManager) Get(ctx context.Context, name string) (*model.Parameter, error) {
	value, err := m.kcm.Get(ctx, parameterKey(name))
	if err != nil {
		return nil, err
	}

	return &model.Parameter{
		Name:  name,
		Value: value,
	}, nil
}

func (m *ParameterManager) Set(ctx context.Context, name string, value interface{}) (*model.Parameter, error) {
	if err := m.kcm.Set(ctx, parameterKey(name), value); err != nil {
		return nil, err
	}

	return &model.Parameter{
		Name:  name,
		Value: value,
	}, nil
}

func NewParameterManager(cm ConfigMap) *ParameterManager {
	return &ParameterManager{
		kcm: NewKVConfigMap(cm),
	}
}

func parameterKey(name string) string {
	return fmt.Sprintf("parameters.%s", name)
}
