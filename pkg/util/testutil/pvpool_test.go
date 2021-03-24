package testutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/util/testutil"
)

func TestInstallPVPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testutil.WithEndToEndEnvironment(t, ctx, nil, func(e2e *testutil.EndToEndEnvironment) {
		testutil.InstallPVPool(t, ctx, e2e.ControllerClient)
	})
}
