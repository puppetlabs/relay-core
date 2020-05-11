package reject

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type specManager struct{}

func (*specManager) Get(ctx context.Context) (*model.Spec, error) {
	return nil, model.ErrRejected
}

func (*specManager) Set(ctx context.Context, value map[string]interface{}) (*model.Spec, error) {
	return nil, model.ErrRejected
}

var SpecManager model.SpecManager = &specManager{}
