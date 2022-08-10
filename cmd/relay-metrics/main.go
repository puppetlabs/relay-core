package main

import (
	"context"
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/metrics/model"
	"github.com/puppetlabs/relay-core/pkg/metrics/opt"
	"github.com/puppetlabs/relay-core/pkg/metrics/reconciler/event"
	"github.com/puppetlabs/relay-core/pkg/metrics/reconciler/run"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	DefaultWorkflowRunPollingInterval = 10 * time.Second
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(
		scheme.AddToScheme,
		corev1.AddToScheme,
		relayv1beta1.AddToScheme,
	)

	Scheme = runtime.NewScheme()
)

func init() {
	if err := SchemeBuilder.AddToScheme(Scheme); err != nil {
		panic(err)
	}
}

func processStatuses(ctx context.Context, c client.Client, meter *metric.Meter) error {
	wrs := &relayv1beta1.RunList{}
	err := c.List(ctx, wrs)
	if err != nil {
		return err
	}

	counter := metric.Must(*meter).NewInt64Counter(model.MetricWorkflowRunCount)
	counter.Add(ctx, int64(len(wrs.Items)))

	for _, wr := range wrs.Items {
		if len(wr.Status.Conditions) == 0 {
			counter.Add(ctx, 1,
				attribute.String(model.MetricAttributeStatus, string(model.WorkflowRunStatusQueued)),
			)

			continue
		}

		for _, cond := range wr.Status.Conditions {
			attrs := []attribute.KeyValue{}

			switch cond.Type {
			case relayv1beta1.RunCompleted:
				switch cond.Status {
				case corev1.ConditionFalse:
					attrs = []attribute.KeyValue{
						attribute.String(model.MetricAttributeStatus, string(model.WorkflowRunStatusInProgress)),
					}
				case corev1.ConditionUnknown:
					attrs = []attribute.KeyValue{
						attribute.String(model.MetricAttributeStatus, string(model.WorkflowRunStatusPending)),
					}
				}
			}

			if len(attrs) > 0 {
				counter := metric.Must(*meter).NewInt64Counter(model.MetricWorkflowRunStatus)
				counter.Add(ctx, 1, attrs...)
			}
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

	err = run.Add(mgr, meter)
	if err != nil {
		klog.Fatal(err.Error())
	}

	err = event.Add(mgr, meter, cfg.EventFilters)
	if err != nil {
		klog.Fatal(err.Error())
	}

	go func() {
		ticker := time.NewTicker(DefaultWorkflowRunPollingInterval)
		defer ticker.Stop()
		for range ticker.C {
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
