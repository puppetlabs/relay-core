package main

import (
	"github.com/inconshreveable/log15"
	"github.com/puppetlabs/horsehead/logging"
)

const (
	defaultAt = "metadata-api"
)

type LoggerOptions struct {
	At    []string
	Debug bool
}

func NewLogger(opts LoggerOptions) logging.Logger {
	lvl := log15.LvlInfo

	if opts.Debug {
		lvl = log15.LvlDebug
	}

	if len(opts.At) == 0 {
		opts.At = []string{defaultAt}
	}

	logging.SetLevel(lvl)

	logger := logging.Builder().At(opts.At...).Build()

	return logger
}
