package authenticate

import (
	"context"

	"github.com/puppetlabs/leg/logging"
)

var (
	logger = logging.Builder().At("relay-core", "pkg", "authenticate")
)

func log(ctx context.Context) logging.Logger {
	return logger.With(ctx).Build()
}
