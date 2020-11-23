package resolve

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type chainDataTypeResolvers struct {
	resolvers []DataTypeResolver
}

func (cr *chainDataTypeResolvers) ResolveData(ctx context.Context, query string) (interface{}, error) {
	for _, r := range cr.resolvers {
		s, err := r.ResolveData(ctx, query)
		if _, ok := err.(*model.DataNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return s, nil
	}

	return "", &model.DataNotFoundError{Query: query}
}

func ChainDataTypeResolvers(resolvers ...DataTypeResolver) DataTypeResolver {
	return &chainDataTypeResolvers{resolvers: resolvers}
}

type chainSecretTypeResolvers struct {
	resolvers []SecretTypeResolver
}

func (cr *chainSecretTypeResolvers) ResolveSecret(ctx context.Context, name string) (string, error) {
	for _, r := range cr.resolvers {
		s, err := r.ResolveSecret(ctx, name)
		if _, ok := err.(*model.SecretNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return s, nil
	}

	return "", &model.SecretNotFoundError{Name: name}
}

func ChainSecretTypeResolvers(resolvers ...SecretTypeResolver) SecretTypeResolver {
	return &chainSecretTypeResolvers{resolvers: resolvers}
}

type chainConnectionTypeResolvers struct {
	resolvers []ConnectionTypeResolver
}

func (cr *chainConnectionTypeResolvers) ResolveConnection(ctx context.Context, connectionType, name string) (interface{}, error) {
	for _, r := range cr.resolvers {
		o, err := r.ResolveConnection(ctx, connectionType, name)
		if _, ok := err.(*model.ConnectionNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return o, nil
	}

	return "", &model.ConnectionNotFoundError{Type: connectionType, Name: name}
}

func ChainConnectionTypeResolvers(resolvers ...ConnectionTypeResolver) ConnectionTypeResolver {
	return &chainConnectionTypeResolvers{resolvers: resolvers}
}

type chainOutputTypeResolvers struct {
	resolvers []OutputTypeResolver
}

func (cr *chainOutputTypeResolvers) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	for _, r := range cr.resolvers {
		o, err := r.ResolveOutput(ctx, from, name)
		if _, ok := err.(*model.OutputNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return o, nil
	}

	return "", &model.OutputNotFoundError{From: from, Name: name}
}

func ChainOutputTypeResolvers(resolvers ...OutputTypeResolver) OutputTypeResolver {
	return &chainOutputTypeResolvers{resolvers: resolvers}
}

type chainParameterTypeResolvers struct {
	resolvers []ParameterTypeResolver
}

func (cr *chainParameterTypeResolvers) ResolveParameter(ctx context.Context, name string) (interface{}, error) {
	for _, r := range cr.resolvers {
		p, err := r.ResolveParameter(ctx, name)
		if _, ok := err.(*model.ParameterNotFoundError); ok {
			continue
		} else if err != nil {
			return nil, err
		}

		return p, nil
	}

	return nil, &model.ParameterNotFoundError{Name: name}
}

func ChainParameterTypeResolvers(resolvers ...ParameterTypeResolver) ParameterTypeResolver {
	return &chainParameterTypeResolvers{resolvers: resolvers}
}

type chainAnswerTypeResolvers struct {
	resolvers []AnswerTypeResolver
}

func (cr *chainAnswerTypeResolvers) ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error) {
	for _, r := range cr.resolvers {
		p, err := r.ResolveAnswer(ctx, askRef, name)
		if _, ok := err.(*model.AnswerNotFoundError); ok {
			continue
		} else if err != nil {
			return nil, err
		}

		return p, nil
	}

	return nil, &model.AnswerNotFoundError{AskRef: askRef, Name: name}
}

func ChainAnswerTypeResolvers(resolvers ...AnswerTypeResolver) AnswerTypeResolver {
	return &chainAnswerTypeResolvers{resolvers: resolvers}
}
