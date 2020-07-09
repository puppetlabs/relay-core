package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ParameterManager struct {
	mut    sync.RWMutex
	params map[string]interface{}
}

var _ model.ParameterManager = &ParameterManager{}

func (m *ParameterManager) Get(ctx context.Context, name string) (*model.Parameter, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	value, found := m.params[name]
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.Parameter{
		Name:  name,
		Value: value,
	}, nil
}

func (m *ParameterManager) Set(ctx context.Context, name string, value interface{}) (*model.Parameter, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.params[name] = value

	return &model.Parameter{
		Name:  name,
		Value: value,
	}, nil
}

type ParameterManagerOption func(pm *ParameterManager)

func ParameterManagerWithInitialParameters(params map[string]interface{}) ParameterManagerOption {
	return func(pm *ParameterManager) {
		for k, v := range params {
			pm.params[k] = v
		}
	}
}

func NewParameterManager(opts ...ParameterManagerOption) *ParameterManager {
	pm := &ParameterManager{
		params: make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(pm)
	}

	return pm
}
