package memory

import (
	"context"
	"fmt"
	"sync"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/decorator"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepDecoratorKey struct {
	StepName, Name string
}

type StepDecoratorMap struct {
	mut        sync.RWMutex
	decorators map[StepDecoratorKey]relayv1beta1.Decorator
}

func (s *StepDecoratorMap) List() []relayv1beta1.Decorator {
	s.mut.RLock()
	defer s.mut.RUnlock()

	decs := []relayv1beta1.Decorator{}

	for _, dec := range s.decorators {
		decs = append(decs, dec)
	}

	return decs
}

func (s *StepDecoratorMap) Set(key StepDecoratorKey, dec relayv1beta1.Decorator) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.decorators[key] = dec
}

func NewStepDecoratorMap() *StepDecoratorMap {
	return &StepDecoratorMap{
		mut:        sync.RWMutex{},
		decorators: make(map[StepDecoratorKey]relayv1beta1.Decorator),
	}
}

type StepDecoratorManager struct {
	me *model.Step
	m  *StepDecoratorMap
}

func (s *StepDecoratorManager) List(ctx context.Context) ([]*model.StepDecorator, error) {
	decs := []*model.StepDecorator{}

	for _, dec := range s.m.List() {
		decs = append(decs, &model.StepDecorator{
			Step:  s.me,
			Name:  dec.Name,
			Value: dec,
		})
	}

	return decs, nil
}

func (s *StepDecoratorManager) Set(ctx context.Context, typ, name string, values map[string]interface{}) error {
	dec := relayv1beta1.Decorator{}

	if err := decorator.DecodeInto(model.DecoratorType(typ), name, values, &dec); err != nil {
		return fmt.Errorf("decorator manager: error decoding values: %w", err)
	}

	s.m.Set(StepDecoratorKey{StepName: s.me.Name, Name: name}, dec)

	return nil
}

var _ model.StepDecoratorManager = &StepDecoratorManager{}

func NewStepDecoratorManager(step *model.Step, backend *StepDecoratorMap) *StepDecoratorManager {
	return &StepDecoratorManager{
		me: step,
		m:  backend,
	}
}
