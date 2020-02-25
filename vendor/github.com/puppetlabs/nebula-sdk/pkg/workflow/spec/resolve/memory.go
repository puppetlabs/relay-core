package resolve

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/fn"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/fnlib"
)

type MemorySecretTypeResolver struct {
	m map[string]string
}

var _ SecretTypeResolver = &MemorySecretTypeResolver{}

func (mr *MemorySecretTypeResolver) ResolveSecret(ctx context.Context, name string) (string, error) {
	s, ok := mr.m[name]
	if !ok {
		return "", &SecretNotFoundError{Name: name}
	}

	return s, nil
}

func NewMemorySecretTypeResolver(m map[string]string) *MemorySecretTypeResolver {
	return &MemorySecretTypeResolver{m: m}
}

type MemoryOutputKey struct {
	From string
	Name string
}

type MemoryOutputTypeResolver struct {
	m map[MemoryOutputKey]interface{}
}

var _ OutputTypeResolver = &MemoryOutputTypeResolver{}

func (mr *MemoryOutputTypeResolver) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	o, ok := mr.m[MemoryOutputKey{From: from, Name: name}]
	if !ok {
		return "", &OutputNotFoundError{From: from, Name: name}
	}

	return o, nil
}

func NewMemoryOutputTypeResolver(m map[MemoryOutputKey]interface{}) *MemoryOutputTypeResolver {
	return &MemoryOutputTypeResolver{m: m}
}

type MemoryParameterTypeResolver struct {
	m map[string]interface{}
}

var _ ParameterTypeResolver = &MemoryParameterTypeResolver{}

func (mr *MemoryParameterTypeResolver) ResolveParameter(ctx context.Context, name string) (interface{}, error) {
	p, ok := mr.m[name]
	if !ok {
		return nil, &ParameterNotFoundError{Name: name}
	}

	return p, nil
}

func NewMemoryParameterTypeResolver(m map[string]interface{}) *MemoryParameterTypeResolver {
	return &MemoryParameterTypeResolver{m: m}
}

type MemoryInvocationResolver struct {
	m fn.Map
}

var _ InvocationResolver = &MemoryInvocationResolver{}

func (mr *MemoryInvocationResolver) ResolveInvocationPositional(ctx context.Context, name string, args []interface{}) (fn.Invoker, error) {
	f, err := mr.m.Descriptor(name)
	if err != nil {
		return nil, &FunctionResolutionError{Name: name, Cause: err}
	}

	i, err := f.PositionalInvoker(args)
	if err != nil {
		return nil, &FunctionResolutionError{Name: name, Cause: err}
	}

	// Wrap invoker so we can add the function name to errors produced while
	// invoking.
	wi := fn.InvokerFunc(func(ctx context.Context) (interface{}, error) {
		r, err := i.Invoke(ctx)
		if err != nil {
			return nil, &FunctionResolutionError{Name: name, Cause: err}
		}

		return r, nil
	})
	return wi, nil
}

func (mr *MemoryInvocationResolver) ResolveInvocation(ctx context.Context, name string, args map[string]interface{}) (fn.Invoker, error) {
	f, err := mr.m.Descriptor(name)
	if err != nil {
		return nil, &FunctionResolutionError{Name: name, Cause: err}
	}

	i, err := f.KeywordInvoker(args)
	if err != nil {
		return nil, &FunctionResolutionError{Name: name, Cause: err}
	}

	// Wrap invoker so we can add the function name to errors produced while
	// invoking.
	wi := fn.InvokerFunc(func(ctx context.Context) (interface{}, error) {
		r, err := i.Invoke(ctx)
		if err != nil {
			return nil, &FunctionResolutionError{Name: name, Cause: err}
		}

		return r, nil
	})
	return wi, nil
}

func NewMemoryInvocationResolver(m fn.Map) *MemoryInvocationResolver {
	return &MemoryInvocationResolver{m: m}
}

func NewDefaultMemoryInvocationResolver() *MemoryInvocationResolver {
	return NewMemoryInvocationResolver(fnlib.Library())
}
