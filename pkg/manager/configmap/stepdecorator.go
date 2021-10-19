package configmap

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/decorator"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepDecoratorManager struct {
	me  *model.Step
	kcm *KVConfigMap
}

var _ model.StepDecoratorManager = &StepDecoratorManager{}

func (m *StepDecoratorManager) List(ctx context.Context) ([]*model.StepDecorator, error) {
	sdm, err := m.kcm.List(ctx, fmt.Sprintf("%s.%s.decorator.", model.ActionTypeStep.Plural, m.me.Hash().HexEncoding()))
	if err != nil {
		return nil, err
	}

	var l []*model.StepDecorator

	for key, value := range sdm {
		obj := v1beta1.Decorator{}
		if err := mapstructure.Decode(value, &obj); err != nil {
			return nil, err
		}

		l = append(l, &model.StepDecorator{
			Step:  m.me,
			Name:  key,
			Value: obj,
		})
	}

	return l, nil
}

func (m *StepDecoratorManager) Set(ctx context.Context, typ, name string, values map[string]interface{}) error {
	if err := m.kcm.Set(ctx, stepNameKey(m.me), m.me.Name); err != nil {
		return err
	}

	dec := v1beta1.Decorator{}

	if err := decorator.DecodeInto(model.DecoratorType(typ), name, values, &dec); err != nil {
		return fmt.Errorf("decorator manager: error decoding values: %w", err)
	}

	if err := m.kcm.Set(ctx, stepDecoratorKey(m.me, dec.Name), dec); err != nil {
		return fmt.Errorf("decorator manager: failed to set decorator key: %w", err)
	}

	return nil
}

func NewStepDecoratorManager(step *model.Step, cm ConfigMap) *StepDecoratorManager {
	return &StepDecoratorManager{
		me:  step,
		kcm: NewKVConfigMap(cm),
	}
}

func stepDecoratorKey(step *model.Step, name string) string {
	return fmt.Sprintf("%s.%s.decorator.%s", step.Type().Plural, step.Hash(), name)
}
