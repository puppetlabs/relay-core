package e2e_test

import (
	"os"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"k8s.io/klog"
)

var e2e *testutil.EndToEndEnvironment

func TestMain(m *testing.M) {
	klog.InitFlags(nil)

	os.Exit(testutil.RunEndToEnd(
		m,
		func(e *testutil.EndToEndEnvironment) {
			e2e = e
		},
		testutil.EndToEndEnvironmentWithTekton,
		testutil.EndToEndEnvironmentWithKnative,
		testutil.EndToEndEnvironmentWithAmbassador,
	))
}
