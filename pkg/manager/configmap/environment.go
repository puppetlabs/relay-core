package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
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

	evs := make(map[string]parse.Tree)

	for name, ev := range value.(map[string]interface{}) {
		evs[name] = parse.Tree(ev)
	}

	return &model.Environment{
		Value: evs,
	}, nil
}

func (m *EnvironmentManager) Set(ctx context.Context, value map[string]interface{}) (*model.Environment, error) {
	if err := m.kcm.Set(ctx, environmentKey(m.me), value); err != nil {
		return nil, err
	}

	evs := make(map[string]parse.Tree)

	for name, ev := range value {
		evs[name] = parse.Tree(ev)
	}

	return &model.Environment{
		Value: evs,
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
