package model

import (
	"context"

	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

type LogManager interface {
	PostLog(ctx context.Context, log *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error)
	PostLogMessage(ctx context.Context, message *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error)
}
