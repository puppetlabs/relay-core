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
	mut            sync.RWMutex
	outputs        map[StepOutputKey]interface{}
	outputMetadata map[StepOutputKey]*model.StepOutputMetadata
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

func (m *StepOutputMap) GetMetadata(key StepOutputKey) (*model.StepOutputMetadata, bool) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	value, found := m.outputMetadata[key]
	return value, found
}

func (m *StepOutputMap) Set(key StepOutputKey, value interface{}) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.outputs[key] = value
}

func (m *StepOutputMap) SetMetadata(key StepOutputKey, metadata *model.StepOutputMetadata) {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.outputMetadata[key] = metadata
}

func NewStepOutputMap() *StepOutputMap {
	return &StepOutputMap{
		outputs:        make(map[StepOutputKey]interface{}),
		outputMetadata: make(map[StepOutputKey]*model.StepOutputMetadata),
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

		so := &model.StepOutput{
			Step: &model.Step{
				Run:  m.me.Run,
				Name: key.StepName,
			},
			Name:  key.Name,
			Value: value,
		}

		if metadata, found := m.m.GetMetadata(key); found {
			so.Metadata = metadata
		}

		l = append(l, so)
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
			so := &model.StepOutput{
				Step:  m.me,
				Name:  key.Name,
				Value: value,
			}

			if metadata, found := m.m.GetMetadata(key); found {
				so.Metadata = metadata
			}

			l = append(l, so)
		}
	}

	return l, nil
}

func (m *StepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	step := &model.Step{
		Run:  m.me.Run,
		Name: stepName,
	}

	key := StepOutputKey{StepName: step.Name, Name: name}

	value, found := m.m.Get(key)
	if !found {
		return nil, model.ErrNotFound
	}

	so := &model.StepOutput{
		Step:  step,
		Name:  name,
		Value: value,
	}

	if metadata, found := m.m.GetMetadata(key); found {
		so.Metadata = metadata
	}

	return so, nil
}

func (m *StepOutputManager) Set(ctx context.Context, name string, value interface{}) error {
	m.m.Set(StepOutputKey{StepName: m.me.Name, Name: name}, value)

	return nil
}

func (m *StepOutputManager) SetMetadata(ctx context.Context, name string, metadata *model.StepOutputMetadata) error {
	m.m.SetMetadata(StepOutputKey{StepName: m.me.Name, Name: name}, metadata)

	return nil
}

func NewStepOutputManager(step *model.Step, backend *StepOutputMap) *StepOutputManager {
	return &StepOutputManager{
		me: step,
		m:  backend,
	}
}
