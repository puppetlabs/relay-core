package reject

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type logManager struct{}

func (*logManager) PostLog(ctx context.Context, value interface{}) ([]byte, error) {
	return nil, model.ErrRejected
}

func (*logManager) PostLogMessage(ctx context.Context, logID string, value interface{}) ([]byte, error) {
	return nil, model.ErrRejected
}

var LogManager model.LogManager = &logManager{}
