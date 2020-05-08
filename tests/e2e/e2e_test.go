package e2e_test

import (
	"os"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
)

var e2e *testutil.EndToEndEnvironment

func TestMain(m *testing.M) {
	os.Exit(testutil.RunEndToEnd(m, func(e *testutil.EndToEndEnvironment) {
		e2e = e
	}, testutil.EndToEndEnvironmentWithTekton, testutil.EndToEndEnvironmentWithKnative))
}
