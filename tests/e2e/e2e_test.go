package e2e_test

import (
	"log"
	"os"
	"testing"

	gometrics "github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"k8s.io/klog"
)

var (
	e2e     *testutil.EndToEndEnvironment
	metrics *gometrics.Metrics
)

func TestMain(m *testing.M) {
	klog.InitFlags(nil)

	var err error
	metrics, err = gometrics.NewNamespace("workflow_controller", gometrics.Options{
		DelegateType:  delegates.NoopDelegate,
		ErrorBehavior: gometrics.ErrorBehaviorLog,
	})
	if err != nil {
		log.Fatalf("failed to set up metrics: %+v", err)
	}

	os.Exit(testutil.RunEndToEnd(
		m,
		func(e *testutil.EndToEndEnvironment) {
			e2e = e
		},
		testutil.EndToEndEnvironmentWithHostpathProvisioner,
		testutil.EndToEndEnvironmentWithTekton,
		testutil.EndToEndEnvironmentWithKnative,
		testutil.EndToEndEnvironmentWithAmbassador,
	))
}
