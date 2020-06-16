package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type SpecManager struct {
	mut sync.RWMutex
	val *model.Spec
}

var _ model.SpecManager = &SpecManager{}

func (m *SpecManager) Get(ctx context.Context) (*model.Spec, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	if m.val == nil {
		return nil, model.ErrNotFound
	}

	return m.val, nil
}

func (m *SpecManager) Set(ctx context.Context, value map[string]interface{}) (*model.Spec, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.val = &model.Spec{
		Tree: parse.Tree(value),
	}

	return m.val, nil
}

type SpecManagerOption func(sm *SpecManager)

func SpecManagerWithInitialSpec(value map[string]interface{}) SpecManagerOption {
	return func(sm *SpecManager) {
		sm.val = &model.Spec{
			Tree: parse.Tree(value),
		}
	}
}

func NewSpecManager(opts ...SpecManagerOption) *SpecManager {
	sm := &SpecManager{}

	for _, opt := range opts {
		opt(sm)
	}

	return sm
}
