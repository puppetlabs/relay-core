package sample

import (
	"github.com/puppetlabs/horsehead/v2/logging"
)

var (
	logger = logging.Builder().At("nebula-tasks", "pkg", "metadataapi", "sample")
)

func log() logging.Logger {
	return logger.Build()
}
