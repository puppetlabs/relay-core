package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type ActionMetadataManager struct {
	mut sync.RWMutex
	val *model.ActionMetadata
}

var _ model.ActionMetadataManager = &ActionMetadataManager{}

func (s *ActionMetadataManager) Get(ctx context.Context) (*model.ActionMetadata, error) {
	return s.val, nil
}

func NewActionMetadataManager(sm *model.ActionMetadata) *ActionMetadataManager {
	return &ActionMetadataManager{
		val: sm,
	}
}
