package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type actionStatusManager struct{}

func (*actionStatusManager) Get(ctx context.Context, action model.Action) (*model.ActionStatus, error) {
	return nil, model.ErrRejected
}

func (*actionStatusManager) Set(ctx context.Context, ss *model.ActionStatus) error {
	return model.ErrRejected
}

var ActionStatusManager model.ActionStatusManager = &actionStatusManager{}
