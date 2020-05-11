package main

import (
	"github.com/puppetlabs/horsehead/v2/logging"
)

var (
	logger = logging.Builder().At("nebula-tasks", "cmd", "relay-metadata-api")
)

func log() logging.Logger {
	return logger.Build()
}
