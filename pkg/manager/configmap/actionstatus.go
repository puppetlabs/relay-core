package configmap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ActionStatusManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.ActionStatusManager = &ActionStatusManager{}

func (m *ActionStatusManager) List(ctx context.Context) ([]*model.ActionStatus, error) {
	sas, err := m.kcm.List(ctx, fmt.Sprintf("%s.", model.ActionTypeStep.Plural))
	if err != nil {
		return nil, err
	}

	var l []*model.ActionStatus

	for key, value := range sas {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 2 || parts[1] != "status" {
			continue
		}

		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		as := &model.ActionStatus{}

		if err := json.Unmarshal(data, as); err != nil {
			return nil, err
		}

		l = append(l, as)
	}

	return l, nil
}

func (m *ActionStatusManager) Get(ctx context.Context, action model.Action) (*model.ActionStatus, error) {
	var ask string
	switch t := action.(type) {
	case *model.Step:
		if stepMe, ok := m.me.(*model.Step); ok {
			ask = actionStatusKey(&model.Step{
				Run:  stepMe.Run,
				Name: t.Name,
			})
		}
	}

	if ask == "" {
		return nil, nil
	}

	encoded, err := m.kcm.Get(ctx, ask)
	if err != nil {
		return nil, err
	}

	as := &model.ActionStatus{}

	data, err := json.Marshal(encoded)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, as); err != nil {
		return nil, err
	}

	return as, nil
}

func (m *ActionStatusManager) Set(ctx context.Context, as *model.ActionStatus) error {
	switch t := m.me.(type) {
	case *model.Step:
		if err := m.kcm.Set(ctx, actionStatusKey(m.me), &model.ActionStatus{
			Name:          t.Name,
			ProcessState:  as.ProcessState,
			WhenCondition: as.WhenCondition,
		}); err != nil {
			return err
		}
	}

	return nil
}

func NewActionStatusManager(action model.Action, cm ConfigMap) *ActionStatusManager {
	return &ActionStatusManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func actionStatusKey(action model.Action) string {
	switch action.Type() {
	case model.ActionTypeStep:
		return fmt.Sprintf("%s.%s.status", action.Type().Plural, action.Hash())
	}

	return ""
}
