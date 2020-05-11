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
	Get(ctx context.Context, stepName, name string) (*StepOutput, error)
}

type StepOutputSetterManager interface {
	Set(ctx context.Context, name string, value interface{}) (*StepOutput, error)
}

type StepOutputManager interface {
	StepOutputGetterManager
	StepOutputSetterManager
}
