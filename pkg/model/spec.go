package model

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/parse"
)

type Spec struct {
	Tree parse.Tree
}

type SpecGetterManager interface {
	// Get retrieves the spec for this action, if any.
	Get(ctx context.Context) (*Spec, error)
}

type SpecSetterManager interface {
	// Set stores the spec for this action.
	Set(ctx context.Context, spec map[string]interface{}) (*Spec, error)
}

type SpecManager interface {
	SpecGetterManager
	SpecSetterManager
}
