package testutil

import (
	"path"
	"path/filepath"
	"runtime"
)

func getFixtures(pattern string) ([]string, error) {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	return filepath.Glob(path.Join(base, pattern))
}
