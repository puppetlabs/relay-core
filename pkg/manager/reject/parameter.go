package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type parameterManager struct{}

func (*parameterManager) List(ctx context.Context) ([]*model.Parameter, error) {
	return nil, model.ErrRejected
}

func (*parameterManager) Get(ctx context.Context, name string) (*model.Parameter, error) {
	return nil, model.ErrRejected
}

func (*parameterManager) Set(ctx context.Context, name string, value interface{}) (*model.Parameter, error) {
	return nil, model.ErrRejected
}

var ParameterManager model.ParameterManager = &parameterManager{}
