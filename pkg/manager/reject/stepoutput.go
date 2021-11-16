package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type stepOutputManager struct{}

func (*stepOutputManager) List(ctx context.Context) ([]*model.StepOutput, error) {
	return nil, model.ErrRejected
}

func (*stepOutputManager) ListSelf(ctx context.Context) ([]*model.StepOutput, error) {
	return nil, model.ErrRejected
}

func (*stepOutputManager) Get(ctx context.Context, stepName, name string) (*model.StepOutput, error) {
	return nil, model.ErrRejected
}

func (*stepOutputManager) Set(ctx context.Context, name string, value interface{}) error {
	return model.ErrRejected
}

func (*stepOutputManager) SetMetadata(ctx context.Context, name string, metadata *model.StepOutputMetadata) error {
	return model.ErrRejected
}

var StepOutputManager model.StepOutputManager = &stepOutputManager{}
