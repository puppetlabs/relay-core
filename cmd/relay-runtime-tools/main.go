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
)

var (
	ep = flag.String("entrypoint", "", "Original specified entrypoint to execute")
)

func main() {
	flag.Parse()

	e := entrypoint.Entrypointer{
		Entrypoint: *ep,
		Args:       flag.Args(),
		Runner:     &realRunner{},
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
