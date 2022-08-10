package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ConditionManager struct {
	mut sync.RWMutex
	val *model.Condition
}

var _ model.ConditionManager = &ConditionManager{}

func (m *ConditionManager) Get(ctx context.Context) (*model.Condition, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	if m.val == nil {
		return nil, model.ErrNotFound
	}

	return m.val, nil
}

func (m *ConditionManager) Set(ctx context.Context, value interface{}) (*model.Condition, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.val = &model.Condition{
		Tree: value,
	}

	return m.val, nil
}

type ConditionManagerOption func(cm *ConditionManager)

func ConditionManagerWithInitialCondition(value interface{}) ConditionManagerOption {
	return func(cm *ConditionManager) {
		cm.val = &model.Condition{
			Tree: value,
		}
	}
}

func NewConditionManager(opts ...ConditionManagerOption) *ConditionManager {
	cm := &ConditionManager{}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}
