package reject

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

type secretManager struct{}

func (*secretManager) Get(ctx context.Context, name string) (*model.Secret, error) {
	return nil, model.ErrRejected
}

var SecretManager model.SecretManager = &secretManager{}
