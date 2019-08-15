package main

import (
	"bytes"
	"log"
	"os"

	"github.com/puppetlabs/nebula-tasks/pkg/cmd"
)

func main() {
	command, err := cmd.NewRootCommand()
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)
	command.SetOut(buf)

	if err := command.Execute(); err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(buf.Bytes())
	os.Exit(0)
}
