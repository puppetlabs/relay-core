package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type actionMetadataManager struct{}

func (*actionMetadataManager) Get(ctx context.Context) (*model.ActionMetadata, error) {
	return nil, model.ErrRejected
}

var ActionMetadataManager model.ActionMetadataManager = &actionMetadataManager{}
