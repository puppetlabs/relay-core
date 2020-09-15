package model

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

type Environment struct {
	Value map[string]parse.Tree
}

type EnvironmentGetterManager interface {
	// Get retrieves the environment for this action, if any.
	Get(ctx context.Context) (*Environment, error)
}

type EnvironmentSetterManager interface {
	// Set stores the environment for this action.
	Set(ctx context.Context, value map[string]interface{}) (*Environment, error)
}

type EnvironmentManager interface {
	EnvironmentGetterManager
	EnvironmentSetterManager
}
