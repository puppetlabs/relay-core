package testutil

import (
	"path"
	"path/filepath"
)

func getFixtures(pattern string) ([]string, error) {
	return filepath.Glob(path.Join(ModuleDirectory, "pkg/util/testutil", pattern))
}
