package configmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ActionStatusManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.ActionStatusManager = &ActionStatusManager{}

func (m *ActionStatusManager) Get(ctx context.Context, action model.Action) (*model.ActionStatus, error) {
	ask := actionStatusKey(action)
	if ask == "" {
		return nil, nil
	}

	encoded, err := m.kcm.Get(ctx, actionStatusKey(m.me))
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
	ask := actionStatusKey(m.me)
	if ask == "" {
		return nil
	}

	if err := m.kcm.Set(ctx, actionStatusKey(m.me), as); err != nil {
		return err
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
	switch action.Type().Singular {
	case model.ActionTypeStep.Singular:
		return fmt.Sprintf("%s.%s.status", action.Type().Plural, action.Hash())
	}

	return ""
}
