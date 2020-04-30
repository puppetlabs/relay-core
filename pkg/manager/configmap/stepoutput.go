package configmap

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type StepOutputManager struct {
	me  *model.Step
	kcm *KVConfigMap
}

var _ model.StepOutputManager = &StepOutputManager{}

func (m *StepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	step := &model.Step{
		Run:  m.me.Run,
		Name: stepName,
	}

	value, err := m.kcm.Get(ctx, stepOutputKey(step, name))
	if err != nil {
		return nil, err
	}

	return &model.StepOutput{
		Step:  step,
		Name:  name,
		Value: value,
	}, nil
}

func (m *StepOutputManager) Set(ctx context.Context, name string, value interface{}) (*model.StepOutput, error) {
	if err := m.kcm.Set(ctx, stepOutputKey(m.me, name), value); err != nil {
		return nil, err
	}

	return &model.StepOutput{
		Step:  m.me,
		Name:  name,
		Value: value,
	}, nil
}

func NewStepOutputManager(step *model.Step, cm ConfigMap) *StepOutputManager {
	return &StepOutputManager{
		me:  step,
		kcm: NewKVConfigMap(cm),
	}
}

func stepOutputKey(step *model.Step, name string) string {
	return fmt.Sprintf("%s/%s/output/%s", step.Type().Plural, step.Hash(), name)
}
