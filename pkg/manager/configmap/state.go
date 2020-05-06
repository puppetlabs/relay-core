package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type StateManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.StateManager = &StateManager{}

func (m *StateManager) Get(ctx context.Context, name string) (*model.State, error) {
	value, err := m.kcm.Get(ctx, stateKey(m.me, name))
	if err != nil {
		return nil, err
	}

	return &model.State{
		Name:  name,
		Value: value,
	}, nil
}

func (m *StateManager) Set(ctx context.Context, name string, value interface{}) (*model.State, error) {
	if err := m.kcm.Set(ctx, stateKey(m.me, name), value); err != nil {
		return nil, err
	}

	return &model.State{
		Name:  name,
		Value: value,
	}, nil
}

func NewStateManager(action model.Action, cm ConfigMap) *StateManager {
	return &StateManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func stateKey(action model.Action, name string) string {
	return fmt.Sprintf("%s.%s.state.%s", action.Type().Plural, action.Hash(), name)
}
