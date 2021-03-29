package configmap

import (
	"context"
	"fmt"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type TimerManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.TimerManager = &TimerManager{}

func (m *TimerManager) Get(ctx context.Context, name string) (*model.Timer, error) {
	value, err := m.kcm.Get(ctx, timerKey(m.me, name))
	if err != nil {
		return nil, err
	}

	ts, ok := value.(string)
	if !ok {
		return nil, model.ErrNotFound
	}

	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, model.ErrNotFound
	}

	return &model.Timer{
		Name: name,
		Time: t,
	}, nil
}

func (m *TimerManager) Set(ctx context.Context, name string, t time.Time) (*model.Timer, error) {
	if ok, err := m.kcm.Insert(ctx, timerKey(m.me, name), t.Format(time.RFC3339)); err != nil {
		return nil, err
	} else if !ok {
		return nil, model.ErrConflict
	}

	return &model.Timer{
		Name: name,
		Time: t,
	}, nil
}

func NewTimerManager(action model.Action, cm ConfigMap) *TimerManager {
	return &TimerManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func timerKey(action model.Action, name string) string {
	return fmt.Sprintf("%s.%s.timers.%s", action.Type().Plural, action.Hash(), name)
}
