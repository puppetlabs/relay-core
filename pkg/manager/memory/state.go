package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StateManager struct {
	mut   sync.RWMutex
	state map[string]interface{}
}

var _ model.StateManager = &StateManager{}

func (m *StateManager) Get(ctx context.Context, name string) (*model.State, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	value, found := m.state[name]
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.State{
		Name:  name,
		Value: value,
	}, nil
}

func (m *StateManager) Set(ctx context.Context, name string, value interface{}) (*model.State, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.state[name] = value

	return &model.State{
		Name:  name,
		Value: value,
	}, nil
}

type StateManagerOption func(sm *StateManager)

func StateManagerWithInitialState(state map[string]interface{}) StateManagerOption {
	return func(sm *StateManager) {
		for k, v := range state {
			sm.state[k] = v
		}
	}
}

func NewStateManager(opts ...StateManagerOption) *StateManager {
	sm := &StateManager{
		state: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(sm)
	}

	return sm
}
