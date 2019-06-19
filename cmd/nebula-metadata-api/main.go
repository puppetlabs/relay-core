package main

import (
	"os"

	"github.com/puppetlabs/insights-stdlib/mainutil"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/data/metadata"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/server"
)

func main() {
	cfg := &config.Config{}
	sec := secrets.New(cfg)
	md := metadata.New(cfg)

	srv := server.New(cfg, sec, md)

	os.Exit(mainutil.TrapAndWait(ctx, srv))
}
