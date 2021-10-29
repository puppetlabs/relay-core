package model

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
)

type DecoratorType string

const (
	// DecoratorTypeLink is a reference to a URI. This informs a UI to display
	// the decorator as a link.
	DecoratorTypeLink DecoratorType = "link"
)

type StepDecorator struct {
	Step  *Step
	Name  string
	Value relayv1beta1.Decorator
}

type StepDecoratorManager interface {
	List(ctx context.Context) ([]*StepDecorator, error)
	Set(ctx context.Context, typ, name string, value map[string]interface{}) error
}
