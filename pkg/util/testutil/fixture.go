package testutil

import (
	"fmt"
	"os"
	"path/filepath"
)

var ModuleDirectory string

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			ModuleDirectory = dir
			return
		}

		next := filepath.Dir(dir)
		if dir == next {
			panic(fmt.Errorf("could not detect go.mod for module root"))
		}

		dir = next
	}
}
