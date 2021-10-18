package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepDecoratorKey struct {
	StepName, Name string
}

type StepDecoratorMap struct {
	mut        sync.RWMutex
	decorators map[StepDecoratorKey]interface{}
}

func (s *StepDecoratorMap) List() []map[string]interface{} {
	s.mut.RLock()
	defer s.mut.RUnlock()

	decs := []map[string]interface{}{}

	for _, dec := range s.decorators {
		v, ok := dec.(map[string]interface{})
		if !ok {
			panic("memory: invalid decorator structure")
		}

		decs = append(decs, v)
	}

	return decs
}

func (s *StepDecoratorMap) Set(key StepDecoratorKey, value map[string]interface{}) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.decorators[key] = value
}

func NewStepDecoratorMap() *StepDecoratorMap {
	return &StepDecoratorMap{
		mut:        sync.RWMutex{},
		decorators: make(map[StepDecoratorKey]interface{}),
	}
}

type StepDecoratorManager struct {
	me *model.Step
	m  *StepDecoratorMap
}

func (s *StepDecoratorManager) List(ctx context.Context) ([]*model.StepDecorator, error) {
	all := s.m.List()

	decs := []*model.StepDecorator{}

	for _, v := range all {
		decs = append(decs, &model.StepDecorator{
			Step:  s.me,
			Name:  v["name"].(string),
			Value: v,
		})
	}

	return decs, nil
}

func (s *StepDecoratorManager) Set(ctx context.Context, value map[string]interface{}) error {
	name, ok := value["name"].(string)
	if !ok {
		return model.ErrNotFound
	}

	s.m.Set(StepDecoratorKey{StepName: s.me.Name, Name: name}, value)

	return nil
}

var _ model.StepDecoratorManager = &StepDecoratorManager{}

func NewStepDecoratorManager(step *model.Step, backend *StepDecoratorMap) *StepDecoratorManager {
	return &StepDecoratorManager{
		me: step,
		m:  backend,
	}
}
