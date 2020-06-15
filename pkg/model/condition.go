package model

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/parse"
)

type Condition struct {
	Tree parse.Tree
}

type ConditionGetterManager interface {
	// Get retrieves the condition scoped to the authenticated action, if any.
	Get(ctx context.Context) (*Condition, error)
}

type ConditionSetterManager interface {
	// Set sets the condition scoped to the authenticated action.
	Set(ctx context.Context, value interface{}) (*Condition, error)
}

type ConditionManager interface {
	ConditionGetterManager
	ConditionSetterManager
}
