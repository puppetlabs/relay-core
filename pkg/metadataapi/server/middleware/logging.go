package middleware

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/logging"
)

var (
	logger = logging.Builder().At("nebula-tasks", "pkg", "metadataapi", "server", "middleware")
)

func log(ctx context.Context) logging.Logger {
	return logger.With(ctx).Build()
}
