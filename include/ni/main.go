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
		log.Fatal(err)
	}

	os.Exit(0)
}
