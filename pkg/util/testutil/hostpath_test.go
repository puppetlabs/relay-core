package testutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/util/testutil"
)

func TestInstallHostpathProvisioner(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	testutil.WithEndToEndEnvironment(t, func(e2e *testutil.EndToEndEnvironment) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		testutil.InstallHostpathProvisioner(t, ctx, e2e.ControllerRuntimeClient)
	})
}
