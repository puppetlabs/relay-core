package configmap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepOutputManager struct {
	me  *model.Step
	kcm *KVConfigMap
}

var _ model.StepOutputManager = &StepOutputManager{}

func (m *StepOutputManager) List(ctx context.Context) ([]*model.StepOutput, error) {
	som, err := m.kcm.List(ctx, fmt.Sprintf("%s.", model.ActionTypeStep.Plural))
	if err != nil {
		return nil, err
	}

	var l []*model.StepOutput

	for key, value := range som {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 || parts[1] != "output" {
			continue
		}

		stepHash, name := parts[0], parts[2]

		stepNameRaw, err := m.kcm.Get(ctx, fmt.Sprintf("%s.%s.name", model.ActionTypeStep.Plural, stepHash))
		if err == model.ErrNotFound {
			continue
		} else if err != nil {
			return nil, err
		}

		stepName, ok := stepNameRaw.(string)
		if !ok {
			continue
		}

		so := &model.StepOutput{
			Step: &model.Step{
				Run:  m.me.Run,
				Name: stepName,
			},
			Name:  name,
			Value: value,
		}

		metadata, err := m.lookupStepOutputMetadata(ctx, so)
		if err != nil {
			return nil, err
		}

		so.Metadata = metadata

		l = append(l, so)
	}

	return l, nil
}

func (m *StepOutputManager) ListSelf(ctx context.Context) ([]*model.StepOutput, error) {
	som, err := m.kcm.List(ctx, fmt.Sprintf("%s.%s.output.", model.ActionTypeStep.Plural, m.me.Hash().HexEncoding()))
	if err != nil {
		return nil, err
	}

	var l []*model.StepOutput

	for key, value := range som {
		so := &model.StepOutput{
			Step:  m.me,
			Name:  key,
			Value: value,
		}

		metadata, err := m.lookupStepOutputMetadata(ctx, so)
		if err != nil {
			return nil, err
		}

		so.Metadata = metadata

		l = append(l, so)
	}

	return l, nil
}

func (m *StepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	step := &model.Step{
		Run:  m.me.Run,
		Name: stepName,
	}

	value, err := m.kcm.Get(ctx, stepOutputKey(step, name))
	if err != nil {
		return nil, err
	}

	so := &model.StepOutput{
		Step:  step,
		Name:  name,
		Value: value,
	}

	metadata, err := m.lookupStepOutputMetadata(ctx, so)
	if err != nil {
		return nil, err
	}

	so.Metadata = metadata

	return so, nil
}

func (m *StepOutputManager) Set(ctx context.Context, name string, value interface{}) error {
	// TODO: Should this be somewhere else? We only need it for the reverse
	// lookup in the list method but it could be useful to other managers down
	// the line.
	if err := m.kcm.Set(ctx, stepNameKey(m.me), m.me.Name); err != nil {
		return err
	}

	if err := m.kcm.Set(ctx, stepOutputKey(m.me, name), value); err != nil {
		return err
	}

	return nil
}

func (m *StepOutputManager) SetMetadata(ctx context.Context, name string, metadata *model.StepOutputMetadata) error {
	if err := m.kcm.Set(ctx, stepOutputMetadataKey(m.me, name), metadata); err != nil {
		return err
	}

	return nil
}

func (m *StepOutputManager) lookupStepOutputMetadata(ctx context.Context, so *model.StepOutput) (*model.StepOutputMetadata, error) {
	data, err := m.kcm.Get(ctx, stepOutputMetadataKey(so.Step, so.Name))
	if err == model.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	metadata := &model.StepOutputMetadata{}
	err = json.Unmarshal(b, metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func NewStepOutputManager(step *model.Step, cm ConfigMap) *StepOutputManager {
	return &StepOutputManager{
		me:  step,
		kcm: NewKVConfigMap(cm),
	}
}

func stepNameKey(step *model.Step) string {
	return fmt.Sprintf("%s.%s.name", step.Type().Plural, step.Hash())
}

func stepOutputKey(step *model.Step, name string) string {
	return fmt.Sprintf("%s.%s.output.%s", step.Type().Plural, step.Hash(), name)
}

func stepOutputMetadataKey(step *model.Step, name string) string {
	return fmt.Sprintf("%s.%s.metadata.output.%s", step.Type().Plural, step.Hash(), name)
}
