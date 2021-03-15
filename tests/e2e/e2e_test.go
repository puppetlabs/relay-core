package e2e_test

import (
	"flag"
	"fmt"

	gometrics "github.com/puppetlabs/leg/instrumentation/metrics"
	"github.com/puppetlabs/leg/instrumentation/metrics/delegates"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var metrics *gometrics.Metrics

func init() {
	kfs := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kfs)
	_ = kfs.Set("v", "5")

	log.SetLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))

	var err error
	metrics, err = gometrics.NewNamespace("workflow_controller", gometrics.Options{
		DelegateType:  delegates.NoopDelegate,
		ErrorBehavior: gometrics.ErrorBehaviorLog,
	})
	if err != nil {
		panic(fmt.Errorf("failed to set up metrics: %w", err))
	}
}
