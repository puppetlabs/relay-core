package model

import "context"

type StepMetadata struct {
	Image string
}

type StepMetadataManager interface {
	Get(ctx context.Context) (*StepMetadata, error)
}
