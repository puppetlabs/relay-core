package model

import (
	"context"
)

type Condition struct {
	Tree any
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
