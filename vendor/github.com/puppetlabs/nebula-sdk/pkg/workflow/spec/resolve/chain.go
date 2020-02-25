package resolve

import "context"

type chainSecretTypeResolvers struct {
	resolvers []SecretTypeResolver
}

func (cr *chainSecretTypeResolvers) ResolveSecret(ctx context.Context, name string) (string, error) {
	for _, r := range cr.resolvers {
		s, err := r.ResolveSecret(ctx, name)
		if _, ok := err.(*SecretNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return s, nil
	}

	return "", &SecretNotFoundError{Name: name}
}

func ChainSecretTypeResolvers(resolvers ...SecretTypeResolver) SecretTypeResolver {
	return &chainSecretTypeResolvers{resolvers: resolvers}
}

type chainOutputTypeResolvers struct {
	resolvers []OutputTypeResolver
}

func (cr *chainOutputTypeResolvers) ResolveOutput(ctx context.Context, from, name string) (interface{}, error) {
	for _, r := range cr.resolvers {
		o, err := r.ResolveOutput(ctx, from, name)
		if _, ok := err.(*OutputNotFoundError); ok {
			continue
		} else if err != nil {
			return "", err
		}

		return o, nil
	}

	return "", &OutputNotFoundError{From: from, Name: name}
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
		if _, ok := err.(*ParameterNotFoundError); ok {
			continue
		} else if err != nil {
			return nil, err
		}

		return p, nil
	}

	return nil, &ParameterNotFoundError{Name: name}
}

func ChainParameterTypeResolvers(resolvers ...ParameterTypeResolver) ParameterTypeResolver {
	return &chainParameterTypeResolvers{resolvers: resolvers}
}
