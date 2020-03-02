package resolve

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/fn"
)

type SecretTypeResolver interface {
	ResolveSecret(ctx context.Context, name string) (string, error)
}

type SecretTypeResolverFunc func(ctx context.Context, name string) (string, error)

var _ SecretTypeResolver = SecretTypeResolverFunc(nil)

func (f SecretTypeResolverFunc) ResolveSecret(ctx context.Context, name string) (string, error) {
	return f(ctx, name)
}

type OutputTypeResolver interface {
	ResolveOutput(ctx context.Context, from, name string) (interface{}, error)
}

type OutputTypeResolverFunc func(ctx context.Context, from, name string) (interface{}, error)

var _ OutputTypeResolver = OutputTypeResolverFunc(nil)

func (f OutputTypeResolverFunc) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	return f(ctx, from, name)
}

type ParameterTypeResolver interface {
	ResolveParameter(ctx context.Context, name string) (interface{}, error)
}

type ParameterTypeResolverFunc func(ctx context.Context, name string) (interface{}, error)

var _ ParameterTypeResolver = ParameterTypeResolverFunc(nil)

func (f ParameterTypeResolverFunc) ResolveParameter(ctx context.Context, name string) (interface{}, error) {
	return f(ctx, name)
}

type AnswerTypeResolver interface {
	ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error)
}

type AnswerTypeResolverFunc func(ctx context.Context, askRef, name string) (interface{}, error)

var _ AnswerTypeResolver = AnswerTypeResolverFunc(nil)

func (f AnswerTypeResolverFunc) ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error) {
	return f(ctx, askRef, name)
}

type InvocationResolver interface {
	ResolveInvocationPositional(ctx context.Context, name string, args []interface{}) (fn.Invoker, error)
	ResolveInvocation(ctx context.Context, name string, args map[string]interface{}) (fn.Invoker, error)
}
