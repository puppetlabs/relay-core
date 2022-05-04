package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ActionStatusKey struct {
	ActionHash string
}

type ActionStatusMap struct {
	mut    sync.RWMutex
	status map[ActionStatusKey]*model.ActionStatus
}

func (m *ActionStatusMap) Keys() []ActionStatusKey {
	m.mut.RLock()
	defer m.mut.RUnlock()

	var l []ActionStatusKey

	for k := range m.status {
		l = append(l, k)
	}

	return l
}

func (m *ActionStatusMap) Get(key ActionStatusKey) (*model.ActionStatus, bool) {
	m.mut.Lock()
	defer m.mut.Unlock()

	value, found := m.status[key]

	return value, found
}

func (m *ActionStatusMap) Set(key ActionStatusKey, message *model.ActionStatus) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.status[key] = message
}

func NewActionStatusMap() *ActionStatusMap {
	return &ActionStatusMap{
		mut:    sync.RWMutex{},
		status: make(map[ActionStatusKey]*model.ActionStatus),
	}
}

type ActionStatusManager struct {
	me model.Action
	m  *ActionStatusMap
}

func (m *ActionStatusManager) List(ctx context.Context) ([]*model.ActionStatus, error) {
	var l []*model.ActionStatus

	for _, key := range m.m.Keys() {
		value, found := m.m.Get(key)
		if !found {
			continue
		}

		switch t := m.me.(type) {
		case *model.Step:
			as := &model.ActionStatus{
				Name:          t.Name,
				ProcessState:  value.ProcessState,
				WhenCondition: value.WhenCondition,
			}

			l = append(l, as)
		}
	}

	return l, nil
}

func (m *ActionStatusManager) Get(ctx context.Context, action model.Action) (*model.ActionStatus, error) {
	value, found := m.m.Get(ActionStatusKey{ActionHash: action.Hash().String()})
	if !found {
		return nil, model.ErrNotFound
	}

	return value, nil
}

func (m *ActionStatusManager) Set(ctx context.Context, sm *model.ActionStatus) error {
	m.m.Set(ActionStatusKey{ActionHash: m.me.Hash().String()}, sm)
	return nil
}

var _ model.ActionStatusManager = &ActionStatusManager{}

func NewActionStatusManager(action model.Action, backend *ActionStatusMap) *ActionStatusManager {
	return &ActionStatusManager{
		me: action,
		m:  backend,
	}
}
