package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type ConditionManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.ConditionManager = &ConditionManager{}

func (m *ConditionManager) Get(ctx context.Context) (*model.Condition, error) {
	value, err := m.kcm.Get(ctx, conditionKey(m.me))
	if err != nil {
		return nil, err
	}

	return &model.Condition{
		Tree: parse.Tree(value),
	}, nil
}

func (m *ConditionManager) Set(ctx context.Context, value interface{}) (*model.Condition, error) {
	if err := m.kcm.Set(ctx, conditionKey(m.me), value); err != nil {
		return nil, err
	}

	return &model.Condition{
		Tree: parse.Tree(value),
	}, nil
}

func NewConditionManager(action model.Action, cm ConfigMap) *ConditionManager {
	return &ConditionManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func conditionKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.condition", action.Type().Plural, action.Hash())
}
