package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type stepMetadataManager struct{}

func (*stepMetadataManager) Get(ctx context.Context) (*model.StepMetadata, error) {
	return nil, model.ErrRejected
}

func (*stepMetadataManager) Set(ctx context.Context, sm *model.StepMetadata) error {
	return model.ErrRejected
}

var StepMetadataManager model.StepMetadataManager = &stepMetadataManager{}
