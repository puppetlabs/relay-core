package metrics

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/logging"
)

var (
	defaultLogger = logging.Builder().At("horsehead", "instrumentation", "metrics")
)

func log(ctx context.Context) logging.Logger {
	return defaultLogger.With(ctx).Build()
}
