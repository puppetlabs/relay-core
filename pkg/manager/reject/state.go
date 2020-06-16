package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type stateManager struct{}

func (*stateManager) Get(ctx context.Context, name string) (*model.State, error) {
	return nil, model.ErrRejected
}

func (*stateManager) Set(ctx context.Context, name string, value interface{}) (*model.State, error) {
	return nil, model.ErrRejected
}

var StateManager model.StateManager = &stateManager{}
