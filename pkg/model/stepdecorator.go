package model

import "context"

type DecoratorType string

const (
	// DecoratorTypeLink is a reference to a URI. This informs a UI to display
	// the decorator as a link.
	DecoratorTypeLink DecoratorType = "link"
)

type StepDecorator struct {
	Step  *Step
	Name  string
	Value interface{}
}

type StepDecoratorManager interface {
	List(ctx context.Context) ([]*StepDecorator, error)
	Set(ctx context.Context, value map[string]interface{}) error
}
