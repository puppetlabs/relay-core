package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepOutputKey struct {
	StepName, Name string
}

type StepOutputMap struct {
	mut     sync.RWMutex
	outputs map[StepOutputKey]interface{}
}

func (m *StepOutputMap) Keys() []StepOutputKey {
	m.mut.RLock()
	defer m.mut.RUnlock()

	var l []StepOutputKey

	for k := range m.outputs {
		l = append(l, k)
	}

	return l
}

func (m *StepOutputMap) Get(key StepOutputKey) (interface{}, bool) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	value, found := m.outputs[key]
	return value, found
}

func (m *StepOutputMap) Set(key StepOutputKey, value interface{}) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.outputs[key] = value
}

func NewStepOutputMap() *StepOutputMap {
	return &StepOutputMap{
		outputs: make(map[StepOutputKey]interface{}),
	}
}

type StepOutputManager struct {
	me *model.Step
	m  *StepOutputMap
}

var _ model.StepOutputManager = &StepOutputManager{}

func (m *StepOutputManager) List(ctx context.Context) ([]*model.StepOutput, error) {
	var l []*model.StepOutput

	for _, key := range m.m.Keys() {
		value, found := m.m.Get(key)
		if !found {
			continue
		}

		l = append(l, &model.StepOutput{
			Step: &model.Step{
				Run:  m.me.Run,
				Name: key.StepName,
			},
			Name:  key.Name,
			Value: value,
		})
	}

	return l, nil
}

func (m *StepOutputManager) ListSelf(ctx context.Context) ([]*model.StepOutput, error) {
	var l []*model.StepOutput

	for _, key := range m.m.Keys() {
		value, found := m.m.Get(key)
		if !found {
			continue
		}

		if key.StepName == m.me.Name {
			l = append(l, &model.StepOutput{
				Step:  m.me,
				Name:  key.Name,
				Value: value,
			})
		}
	}

	return l, nil
}

func (m *StepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	step := &model.Step{
		Run:  m.me.Run,
		Name: stepName,
	}

	value, found := m.m.Get(StepOutputKey{StepName: step.Name, Name: name})
	if !found {
		return nil, model.ErrNotFound
	}

	return &model.StepOutput{
		Step:  step,
		Name:  name,
		Value: value,
	}, nil
}

func (m *StepOutputManager) Set(ctx context.Context, name string, value interface{}) (*model.StepOutput, error) {
	m.m.Set(StepOutputKey{StepName: m.me.Name, Name: name}, value)

	return &model.StepOutput{
		Step:  m.me,
		Name:  name,
		Value: value,
	}, nil
}

func NewStepOutputManager(step *model.Step, backend *StepOutputMap) *StepOutputManager {
	return &StepOutputManager{
		me: step,
		m:  backend,
	}
}
