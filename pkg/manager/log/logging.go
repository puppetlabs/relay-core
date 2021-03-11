package log

import (
	"context"

	"github.com/puppetlabs/leg/logging"
)

var (
	logger = logging.Builder().At("relay-core", "pkg", "manager", "log")
)

func log(ctx context.Context) logging.Logger {
	return logger.With(ctx).Build()
}
