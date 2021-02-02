package main

import (
	"context"
	"time"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/metrics/model"
	"github.com/puppetlabs/relay-core/pkg/metrics/opt"
	"github.com/puppetlabs/relay-core/pkg/metrics/reconciler"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"knative.dev/pkg/signals"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultWorkflowRunPollingInterval = 10 * time.Second
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(
		scheme.AddToScheme,
		nebulav1.AddToScheme,
	)

	Scheme = runtime.NewScheme()
)

func init() {
	if err := SchemeBuilder.AddToScheme(Scheme); err != nil {
		panic(err)
	}
}

func processStatuses(ctx context.Context, c client.Client, meter *metric.Meter) error {
	wrs := &nebulav1.WorkflowRunList{}
	err := c.List(ctx, wrs)
	if err != nil {
		return err
	}

	counter := metric.Must(*meter).NewInt64Counter(model.MetricWorkflowRunCount)
	counter.Add(context.Background(), int64(len(wrs.Items)))

	for _, wr := range wrs.Items {
		status := wr.Status.Status
		if len(status) == 0 {
			status = string(obj.WorkflowRunStatusQueued)
		}

		switch status {
		case string(obj.WorkflowRunStatusQueued), string(obj.WorkflowRunStatusPending), string(obj.WorkflowRunStatusInProgress):
			counter := metric.Must(*meter).NewInt64Counter(model.MetricWorkflowRunStatus)
			counter.Add(context.Background(), 1,
				label.String(model.MetricLabelStatus, status),
			)
		}
	}

	return nil
}

func main() {
	klog.InitFlags(nil)

	cfg, err := opt.NewConfig()
	if err != nil {
		klog.Fatal(err.Error())
	}

	meter, err := cfg.Metrics()
	if err != nil {
		klog.Fatal(err.Error())
	}

	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{},
		&clientcmd.ConfigOverrides{},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		klog.Fatal(err.Error())
	}

	mgr, err := ctrl.NewManager(kcc, ctrl.Options{
		Scheme: Scheme,
	})
	if err != nil {
		klog.Fatal(err.Error())
	}

	err = reconciler.Add(mgr, meter)
	if err != nil {
		klog.Fatal(err.Error())
	}

	go func() {
		ticker := time.NewTicker(DefaultWorkflowRunPollingInterval)
		defer ticker.Stop()
		for ; true; <-ticker.C {
			err := processStatuses(context.Background(), mgr.GetClient(), meter)
			if err != nil {
				klog.Error(err)
			}
		}
	}()

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatal(err.Error())
	}
}
