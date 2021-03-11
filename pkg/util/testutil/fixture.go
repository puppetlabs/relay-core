package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ModuleDirectory   string
	GoBuildIdentifier string
)

func setUpModuleDirectory() {
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

func setUpGoBuildIdentifier() {
	executable, err := os.Executable()
	if err != nil {
		panic(fmt.Errorf("unable to get executable path for test: %w", err))
	}

	tmp := os.Getenv("GOTMPDIR")
	if tmp == "" {
		tmp = os.TempDir()
	}

	re := regexp.MustCompile("^" + regexp.QuoteMeta(filepath.Join(tmp, "go-build")) + `([^/]+)/`)
	matches := re.FindStringSubmatch(executable)
	if matches == nil {
		panic(fmt.Errorf("executable path %s does not match regular expression %s", executable, re))
	}

	GoBuildIdentifier = matches[1]
}

func init() {
	setUpModuleDirectory()
	setUpGoBuildIdentifier()
}
