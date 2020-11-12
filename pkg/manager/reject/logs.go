package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

type logManager struct{}

func (*logManager) PostLog(ctx context.Context, log *plspb.LogCreateRequest) (*plspb.LogCreateResponse, error) {
	return nil, model.ErrRejected
}

func (*logManager) PostLogMessage(ctx context.Context, message *plspb.LogMessageAppendRequest) (*plspb.LogMessageAppendResponse, error) {
	return nil, model.ErrRejected
}

var LogManager model.LogManager = &logManager{}
