package mainutil

import (
	"context"

	logging "github.com/puppetlabs/horsehead/logging"
)

var (
	logger = logging.Builder().At("horsehead", "mainutil")
)

func log(ctx context.Context) logging.Logger {
	return logger.With(ctx).Build()
}
