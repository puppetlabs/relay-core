package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type stepMessageManager struct{}

func (*stepMessageManager) List(ctx context.Context) ([]*model.StepMessage, error) {
	return nil, model.ErrRejected
}

func (*stepMessageManager) Set(ctx context.Context, sm *model.StepMessage) error {
	return model.ErrRejected
}

var StepMessageManager model.StepMessageManager = &stepMessageManager{}
