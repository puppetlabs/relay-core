package main

import (
	"github.com/puppetlabs/leg/logging"
)

var (
	logger = logging.Builder().At("relay-core", "cmd", "relay-metadata-api")
)

func log() logging.Logger {
	return logger.Build()
}
