package memory

import (
	"context"
	"sync"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type TimerManager struct {
	mut    sync.RWMutex
	timers map[string]time.Time
}

var _ model.TimerManager = &TimerManager{}

func (m *TimerManager) Get(ctx context.Context, name string) (*model.Timer, error) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	t, found := m.timers[name]
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.Timer{
		Name: name,
		Time: t,
	}, nil
}

func (m *TimerManager) Set(ctx context.Context, name string, t time.Time) (*model.Timer, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if _, found := m.timers[name]; found {
		return nil, model.ErrConflict
	}

	m.timers[name] = t

	return &model.Timer{
		Name: name,
		Time: t,
	}, nil
}

func NewTimerManager() *TimerManager {
	return &TimerManager{
		timers: make(map[string]time.Time),
	}
}
