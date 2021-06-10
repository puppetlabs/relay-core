package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type secretManager struct{}

func (*secretManager) List(ctx context.Context) ([]*model.Secret, error) {
	return nil, model.ErrRejected
}

func (*secretManager) Get(ctx context.Context, name string) (*model.Secret, error) {
	return nil, model.ErrRejected
}

var SecretManager model.SecretManager = &secretManager{}
