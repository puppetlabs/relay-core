package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepMessageKey struct {
	StepName, ID string
}

type StepMessageMap struct {
	mut      sync.RWMutex
	messages map[StepMessageKey]*model.StepMessage
}

func (s *StepMessageMap) List(step *model.Step) []*model.StepMessage {
	s.mut.RLock()
	defer s.mut.RUnlock()

	msgs := make([]*model.StepMessage, 0)

	for key, sm := range s.messages {
		if step.Name == key.StepName {
			msgs = append(msgs, sm)
		}
	}

	return msgs
}

func (s *StepMessageMap) Set(key StepMessageKey, message *model.StepMessage) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.messages[key] = message
}

func NewStepMessageMap() *StepMessageMap {
	return &StepMessageMap{
		mut:      sync.RWMutex{},
		messages: make(map[StepMessageKey]*model.StepMessage),
	}
}

type StepMessageManager struct {
	me *model.Step
	m  *StepMessageMap
}

func (s *StepMessageManager) List(ctx context.Context) ([]*model.StepMessage, error) {
	msgs := []*model.StepMessage{}
	msgs = append(msgs, s.m.List(s.me)...)

	return msgs, nil
}

func (s *StepMessageManager) Set(ctx context.Context, sm *model.StepMessage) error {
	s.m.Set(StepMessageKey{StepName: s.me.Name, ID: sm.ID}, sm)

	return nil
}

var _ model.StepMessageManager = &StepMessageManager{}

func NewStepMessageManager(step *model.Step, backend *StepMessageMap) *StepMessageManager {
	return &StepMessageManager{
		me: step,
		m:  backend,
	}
}
