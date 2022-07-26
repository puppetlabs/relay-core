package model

import (
	"context"
)

type Spec struct {
	Tree any
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
