package main

import (
	"log"
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/cmd"
)

func main() {
	command, err := cmd.NewRootCommand()
	if err != nil {
		log.Fatal(err)
	}

	command.SetOut(os.Stdout)
	command.SetErr(os.Stderr)

	if err := command.Execute(); err != nil {
		command.ErrOrStderr().Write([]byte(err.Error()))

		os.Exit(1)
	}

	os.Exit(0)
}
