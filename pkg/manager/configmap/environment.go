package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type EnvironmentManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.EnvironmentManager = &EnvironmentManager{}

func (m *EnvironmentManager) Get(ctx context.Context) (*model.Environment, error) {
	value, err := m.kcm.Get(ctx, environmentKey(m.me))
	if err != nil {
		return nil, err
	}

	return &model.Environment{
		Value: value.(map[string]any),
	}, nil
}

func (m *EnvironmentManager) Set(ctx context.Context, value map[string]any) (*model.Environment, error) {
	if err := m.kcm.Set(ctx, environmentKey(m.me), value); err != nil {
		return nil, err
	}

	return &model.Environment{
		Value: value,
	}, nil
}

func NewEnvironmentManager(action model.Action, cm ConfigMap) *EnvironmentManager {
	return &EnvironmentManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func environmentKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.environment", action.Type().Plural, action.Hash())
}
