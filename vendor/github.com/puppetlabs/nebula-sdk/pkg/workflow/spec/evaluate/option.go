package evaluate

import "github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"

type Option func(e *Evaluator)

func WithSecretTypeResolver(resolver resolve.SecretTypeResolver) Option {
	return func(e *Evaluator) {
		e.secretTypeResolver = resolver
	}
}

func WithOutputTypeResolver(resolver resolve.OutputTypeResolver) Option {
	return func(e *Evaluator) {
		e.outputTypeResolver = resolver
	}
}

func WithParameterTypeResolver(resolver resolve.ParameterTypeResolver) Option {
	return func(e *Evaluator) {
		e.parameterTypeResolver = resolver
	}
}

func WithInvocationResolver(resolver resolve.InvocationResolver) Option {
	return func(e *Evaluator) {
		e.invocationResolver = resolver
	}
}

func WithLanguage(lang Language) Option {
	return func(e *Evaluator) {
		e.lang = lang
	}
}

func WithInvokeFunc(fn InvokeFunc) Option {
	return func(e *Evaluator) {
		e.invoke = fn
	}
}

func WithResultMapper(rm ResultMapper) Option {
	return func(e *Evaluator) {
		e.resultMapper = rm
	}
}
