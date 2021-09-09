package model

import (
	"context"
)

type StepOutput struct {
	Step  *Step
	Name  string
	Value interface{}
}

type StepOutputGetterManager interface {
	List(ctx context.Context) ([]*StepOutput, error)
	ListSelf(ctx context.Context) ([]*StepOutput, error)
	Get(ctx context.Context, stepName, name string) (*StepOutput, error)
}

type StepOutputSetterManager interface {
	Set(ctx context.Context, name string, value interface{}) (*StepOutput, error)
}

type StepOutputManager interface {
	StepOutputGetterManager
	StepOutputSetterManager
}
