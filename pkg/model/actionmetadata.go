package model

import "context"

type ActionMetadata struct {
	Image string
}

type ActionMetadataManager interface {
	Get(ctx context.Context) (*ActionMetadata, error)
}
