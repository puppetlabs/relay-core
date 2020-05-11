package resolve

import (
	"context"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type SecretTypeResolver struct {
	m model.SecretManager
}

var _ resolve.SecretTypeResolver = &SecretTypeResolver{}

func (str *SecretTypeResolver) ResolveSecret(ctx context.Context, name string) (string, error) {
	so, err := str.m.Get(ctx, name)
	if err == model.ErrNotFound {
		return "", &resolve.SecretNotFoundError{Name: name}
	} else if err != nil {
		return "", err
	}

	return so.Value, nil
}

func NewSecretTypeResolver(m model.SecretManager) *SecretTypeResolver {
	return &SecretTypeResolver{
		m: m,
	}
}
