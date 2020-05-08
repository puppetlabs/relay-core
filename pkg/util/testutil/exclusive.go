package testutil

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

type LockID string

const (
	LockEndToEndEnvironment LockID = "end-to-end"
)

func Exclusive(ctx context.Context, name LockID) (release func(), err error) {
	f := exclusiveLockPath(string(name))
	if err := os.MkdirAll(filepath.Dir(f), 0755); err != nil {
		return nil, err
	}

	lock := flock.New(f)

	requested := time.Now()
	if _, err := lock.TryLockContext(ctx, 500*time.Millisecond); err != nil {
		return nil, err
	}

	acquired := time.Now()
	log.Printf("acquired exclusive test lock %q in %s", name, acquired.Sub(requested))

	release = func() {
		if err := lock.Unlock(); err != nil {
			panic(err)
		}

		released := time.Now()
		log.Printf("released exclusive test lock %q after %s (total %s since initial request)", name, released.Sub(acquired), released.Sub(requested))
	}
	return
}

func WithExclusive(ctx context.Context, name LockID, fn func()) error {
	release, err := Exclusive(ctx, name)
	if err != nil {
		return err
	}
	defer release()

	fn()
	return nil
}

func exclusiveLockPath(name string) string {
	return filepath.Join(ModuleDirectory, "tests", ".exclusive", name)
}
