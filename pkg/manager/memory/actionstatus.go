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

func (s *ActionStatusMap) Get(key ActionStatusKey) (*model.ActionStatus, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	return s.status[key], nil
}

func (s *ActionStatusMap) Set(key ActionStatusKey, message *model.ActionStatus) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.status[key] = message
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

func (s *ActionStatusManager) Get(ctx context.Context, action model.Action) (*model.ActionStatus, error) {
	return s.m.Get(ActionStatusKey{ActionHash: action.Hash().String()})
}

func (s *ActionStatusManager) Set(ctx context.Context, sm *model.ActionStatus) error {
	s.m.Set(ActionStatusKey{ActionHash: s.me.Hash().String()}, sm)
	return nil
}

var _ model.ActionStatusManager = &ActionStatusManager{}

func NewActionStatusManager(action model.Action, backend *ActionStatusMap) *ActionStatusManager {
	return &ActionStatusManager{
		me: action,
		m:  backend,
	}
}
