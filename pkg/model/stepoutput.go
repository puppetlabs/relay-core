package model

import (
	"context"
)

type StepOutput struct {
	Step     *Step
	Name     string
	Value    interface{}
	Metadata *StepOutputMetadata
}

type StepOutputMetadata struct {
	Sensitive bool
}

type StepOutputGetterManager interface {
	List(ctx context.Context) ([]*StepOutput, error)
	ListSelf(ctx context.Context) ([]*StepOutput, error)
	Get(ctx context.Context, stepName, name string) (*StepOutput, error)
}

type StepOutputSetterManager interface {
	Set(ctx context.Context, name string, value interface{}) error
	SetMetadata(ctx context.Context, name string, metadata *StepOutputMetadata) error
}

type StepOutputManager interface {
	StepOutputGetterManager
	StepOutputSetterManager
}
