package testutil_test

import (
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	log.SetLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))
}
