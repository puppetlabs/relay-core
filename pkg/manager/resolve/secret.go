package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type SecretTypeResolver struct {
	m model.SecretManager
}

var _ resolve.SecretTypeResolver = &SecretTypeResolver{}

func (str *SecretTypeResolver) ResolveAllSecrets(ctx context.Context) (map[string]string, error) {
	l, err := str.m.List(ctx)
	if err != nil {
		return nil, err
	} else if len(l) == 0 {
		return nil, nil
	}

	sm := make(map[string]string, len(l))

	for _, s := range l {
		sm[s.Name] = s.Value
	}

	return sm, nil
}

func (str *SecretTypeResolver) ResolveSecret(ctx context.Context, name string) (string, error) {
	s, err := str.m.Get(ctx, name)
	if err == model.ErrNotFound {
		return "", &exprmodel.SecretNotFoundError{Name: name}
	} else if err != nil {
		return "", err
	}

	return s.Value, nil
}

func NewSecretTypeResolver(m model.SecretManager) *SecretTypeResolver {
	return &SecretTypeResolver{
		m: m,
	}
}
