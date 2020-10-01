package configmap

import (
	"context"
	"errors"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/model"
)

// StepMetadataManager gets and sets metadata for a step. Right now the model
// only has Image for a value so it's a little naive. This can later be
// expanded to encode the entire StepMetadata struct into a single configmap
// field.
type StepMetadataManager struct {
	me  model.Action
	kcm *KVConfigMap
}

var _ model.StepMetadataManager = &StepMetadataManager{}

func (m *StepMetadataManager) Get(ctx context.Context) (*model.StepMetadata, error) {
	imageValue, err := m.kcm.Get(ctx, imageKey(m.me))
	if err != nil {
		return nil, err
	}

	image, ok := imageValue.(string)
	if !ok {
		return nil, errors.New("invalid type for step image when attempting type assertion")
	}

	return &model.StepMetadata{
		Image: image,
	}, nil
}

func (m *StepMetadataManager) Set(ctx context.Context, sm *model.StepMetadata) error {
	if err := m.kcm.Set(ctx, imageKey(m.me), sm.Image); err != nil {
		return err
	}

	return nil
}

func NewStepMetadataManager(action model.Action, cm ConfigMap) *StepMetadataManager {
	return &StepMetadataManager{
		me:  action,
		kcm: NewKVConfigMap(cm),
	}
}

func imageKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.image", action.Type().Plural, action.Hash())
}
