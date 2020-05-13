package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type SpecManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.SpecManager = &SpecManager{}

func (m *SpecManager) Get(ctx context.Context) (*model.Spec, error) {
	value, err := m.kcm.Get(ctx, specKey(m.me))
	if err != nil {
		return nil, err
	}

	return &model.Spec{
		Tree: parse.Tree(value),
	}, nil
}

func (m *SpecManager) Set(ctx context.Context, value map[string]interface{}) (*model.Spec, error) {
	if err := m.kcm.Set(ctx, specKey(m.me), value); err != nil {
		return nil, err
	}

	return &model.Spec{
		Tree: parse.Tree(value),
	}, nil
}

func NewSpecManager(action model.Action, cm ConfigMap) *SpecManager {
	return &SpecManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func specKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.spec", action.Type().Plural, action.Hash())
}
