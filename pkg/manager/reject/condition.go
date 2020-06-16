package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type conditionManager struct{}

var _ model.ConditionManager = &conditionManager{}

func (m *conditionManager) Get(ctx context.Context) (*model.Condition, error) {
	return nil, model.ErrRejected
}

func (m *conditionManager) Set(ctx context.Context, value interface{}) (*model.Condition, error) {
	return nil, model.ErrRejected
}

var ConditionManager model.ConditionManager = &conditionManager{}
