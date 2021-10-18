package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type stepDecoratorManager struct{}

func (s *stepDecoratorManager) List(ctx context.Context) ([]*model.StepDecorator, error) {
	return nil, model.ErrRejected
}

func (s *stepDecoratorManager) Set(ctx context.Context, value map[string]interface{}) error {
	return model.ErrRejected
}

var StepDecoratorManager model.StepDecoratorManager = &stepDecoratorManager{}
