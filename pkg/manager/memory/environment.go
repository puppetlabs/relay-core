package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type EnvironmentManager struct {
	mut sync.RWMutex
	val *model.Environment
}

var _ model.EnvironmentManager = &EnvironmentManager{}

func (m *EnvironmentManager) Get(ctx context.Context) (*model.Environment, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	if m.val == nil {
		return nil, model.ErrNotFound
	}

	return m.val, nil
}

func (m *EnvironmentManager) Set(ctx context.Context, value map[string]any) (*model.Environment, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.val = &model.Environment{
		Value: value,
	}

	return m.val, nil
}

type EnvironmentManagerOption func(em *EnvironmentManager)

func EnvironmentManagerWithInitialEnvironment(value map[string]any) EnvironmentManagerOption {
	return func(em *EnvironmentManager) {
		em.val = &model.Environment{
			Value: value,
		}
	}
}

func NewEnvironmentManager(opts ...EnvironmentManagerOption) *EnvironmentManager {
	em := &EnvironmentManager{}

	for _, opt := range opts {
		opt(em)
	}

	return em
}
