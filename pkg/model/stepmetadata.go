package model

import "context"

type StepMetadata struct {
	Image string
}

type StepMetadataGetterManager interface {
	Get(ctx context.Context) (*StepMetadata, error)
}

type StepMetadataSetterManager interface {
	Set(ctx context.Context, sm *StepMetadata) error
}

type StepMetadataManager interface {
	StepMetadataGetterManager
	StepMetadataSetterManager
}
