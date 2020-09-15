package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
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

func (m *EnvironmentManager) Set(ctx context.Context, value map[string]interface{}) (*model.Environment, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	evs := make(map[string]parse.Tree)

	for name, ev := range value {
		evs[name] = parse.Tree(ev)
	}

	m.val = &model.Environment{
		Value: evs,
	}

	return m.val, nil
}

type EnvironmentManagerOption func(em *EnvironmentManager)

func EnvironmentManagerWithInitialEnvironment(value map[string]interface{}) EnvironmentManagerOption {
	return func(em *EnvironmentManager) {
		evs := make(map[string]parse.Tree)

		for name, ev := range value {
			evs[name] = parse.Tree(ev)
		}

		em.val = &model.Environment{
			Value: evs,
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
