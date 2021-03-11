package sample

import (
	"github.com/puppetlabs/leg/logging"
)

var (
	logger = logging.Builder().At("relay-core", "pkg", "metadataapi", "sample")
)

func log() logging.Logger {
	return logger.Build()
}
