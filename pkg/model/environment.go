package model

import (
	"context"
)

type Environment struct {
	Value map[string]any
}

type EnvironmentGetterManager interface {
	// Get retrieves the environment for this action, if any.
	Get(ctx context.Context) (*Environment, error)
}

type EnvironmentSetterManager interface {
	// Set stores the environment for this action.
	Set(ctx context.Context, value map[string]any) (*Environment, error)
}

type EnvironmentManager interface {
	EnvironmentGetterManager
	EnvironmentSetterManager
}
