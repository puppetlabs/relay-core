package configmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepMessageManager struct {
	me  *model.Step
	kcm *KVConfigMap
}

var _ model.StepMessageManager = &StepMessageManager{}

func (m *StepMessageManager) List(ctx context.Context) ([]*model.StepMessage, error) {
	som, err := m.kcm.List(ctx, fmt.Sprintf("%s.%s.message.", model.ActionTypeStep.Plural, m.me.Hash().HexEncoding()))
	if err != nil {
		return nil, err
	}

	var l []*model.StepMessage

	for _, encoded := range som {
		sm := &model.StepMessage{}

		data, err := json.Marshal(encoded)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, sm); err != nil {
			return nil, err
		}

		l = append(l, sm)
	}
	return l, nil
}

func (m *StepMessageManager) Set(ctx context.Context, sm *model.StepMessage) error {
	if err := m.kcm.Set(ctx, stepMessageKey(m.me, sm.ID), sm); err != nil {
		return err
	}

	return nil
}

func NewStepMessageManager(step *model.Step, cm ConfigMap) *StepMessageManager {
	return &StepMessageManager{
		me:  step,
		kcm: NewKVConfigMap(cm),
	}
}

func stepMessageKey(step *model.Step, id string) string {
	return fmt.Sprintf("%s.%s.message.%s", step.Type().Plural, step.Hash(), id)
}
