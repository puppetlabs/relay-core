package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepOutputMap struct {
	mut     sync.RWMutex
	outputs map[model.Hash]map[string]interface{}
}

func (m *StepOutputMap) Get(step *model.Step, name string) (interface{}, bool) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	outputs, found := m.outputs[step.Hash()]
	if !found {
		return nil, false
	}

	value, found := outputs[name]
	return value, found
}

func (m *StepOutputMap) Set(step *model.Step, name string, value interface{}) {
	m.mut.Lock()
	defer m.mut.Unlock()

	h := step.Hash()

	outputs, found := m.outputs[h]
	if !found {
		outputs = make(map[string]interface{})
		m.outputs[h] = outputs
	}

	outputs[name] = value
}

func NewStepOutputMap() *StepOutputMap {
	return &StepOutputMap{
		outputs: make(map[model.Hash]map[string]interface{}),
	}
}

type StepOutputManager struct {
	me *model.Step
	m  *StepOutputMap
}

var _ model.StepOutputManager = &StepOutputManager{}

func (m *StepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	step := &model.Step{
		Run:  m.me.Run,
		Name: stepName,
	}

	value, found := m.m.Get(step, name)
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
	m.m.Set(m.me, name, value)

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
