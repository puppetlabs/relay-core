package configmap

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
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

func (m *StepDecoratorManager) Set(ctx context.Context, value map[string]interface{}) error {
	if err := m.kcm.Set(ctx, stepNameKey(m.me), m.me.Name); err != nil {
		return err
	}

	typ, ok := value["type"].(string)
	if !ok {
		return errors.New("decorator manager: missing type field")
	}

	name, ok := value["name"].(string)
	if !ok {
		return errors.New("decorator manager: missing name field")
	}

	dec := v1beta1.Decorator{
		Name: name,
	}

	switch model.DecoratorType(typ) {
	case model.DecoratorTypeLink:
		dl := v1beta1.DecoratorLink{}
		if err := mapstructure.Decode(value, &dl); err != nil {
			return fmt.Errorf("decorator manager: failed to map expected values to decorator: %w", err)
		}

		if _, err := url.Parse(dl.URI); err != nil {
			return fmt.Errorf("decorator manager: failed to parse uri value: %w", err)
		}

		dec.Link = &dl
	default:
		return fmt.Errorf("decorator manager: no such decorator type: %s", typ)
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
