package memory

import (
	"context"
	"sync"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type StepMetadataManager struct {
	mut sync.RWMutex
	val *model.StepMetadata
}

var _ model.StepMetadataManager = &StepMetadataManager{}

func (s *StepMetadataManager) Get(ctx context.Context) (*model.StepMetadata, error) {
	return s.val, nil
}

func (s *StepMetadataManager) Set(ctx context.Context, sm *model.StepMetadata) error {
	s.val = sm
	return nil
}

func NewStepMetadataManager(sm *model.StepMetadata) *StepMetadataManager {
	return &StepMetadataManager{
		val: sm,
	}
}
