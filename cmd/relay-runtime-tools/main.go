// This content has been partially derived from Tekton
// https://github.com/tektoncd/pipeline

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/entrypoint/cmd"
)

var (
	ep = flag.String("entrypoint", "", "Original specified entrypoint to execute")
)

func main() {
	flag.Parse()

	cfg := entrypoint.NewConfig()

	if ep == nil || *ep == "" {
		if len(flag.Args()) > 0 {
			commands := cmd.NewMap()
			if command, ok := commands[flag.Args()[0]]; ok {
				if err := command.Execute(flag.Args()); err != nil {
					log.Fatal(err)
				}
			}
		}

		os.Exit(0)
	}

	e := entrypoint.Entrypointer{
		Entrypoint: *ep,
		Args:       flag.Args(),
		Runner: &entrypoint.RealRunner{
			Config: cfg,
		},
	}

	if err := e.Go(); err != nil {
		switch t := err.(type) {
		case *exec.ExitError:
			// Copied from https://stackoverflow.com/questions/10385551/get-exit-code-go
			// This works on both Unix and Windows. Although
			// package syscall is generally platform dependent,
			// WaitStatus is defined for both Unix and Windows and
			// in both cases has an ExitStatus() method with the
			// same signature.
			if status, ok := t.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
			log.Fatalf("Error executing command (ExitError): %v", err)
		default:
			log.Fatalf("Error executing command: %v", err)
		}
	}
}
