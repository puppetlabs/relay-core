package model

import "context"

type StepDecorator struct {
	Step  *Step
	Name  string
	Value interface{}
}

type StepDecoratorManager interface {
	List(ctx context.Context) ([]*StepDecorator, error)
	Set(ctx context.Context, value map[string]interface{}) error
}
