package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type environmentManager struct{}

func (*environmentManager) Get(ctx context.Context) (*model.Environment, error) {
	return nil, model.ErrRejected
}

func (*environmentManager) Set(ctx context.Context, value map[string]interface{}) (*model.Environment, error) {
	return nil, model.ErrRejected
}

var EnvironmentManager model.EnvironmentManager = &environmentManager{}
