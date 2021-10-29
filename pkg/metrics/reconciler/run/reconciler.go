package run

import (
	"context"
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/metrics/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RunReconciler struct {
	client client.Client
	meter  *metric.Meter
}

var _ reconcile.Reconciler = &RunReconciler{}

func (r *RunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	wr := &relayv1beta1.Run{}

	if err := r.client.Get(ctx, req.NamespacedName, wr); errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if !wr.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	for _, cond := range wr.Status.Conditions {
		attrs := []attribute.KeyValue{}

		switch cond.Type {
		case relayv1beta1.RunCancelled:
			if cond.Status == corev1.ConditionTrue {
				attrs = []attribute.KeyValue{
					attribute.String(model.MetricAttributeOutcome, string(model.WorkflowRunStatusCancelled)),
				}
			}
		case relayv1beta1.RunSucceeded:
			switch cond.Status {
			case corev1.ConditionTrue:
				attrs = []attribute.KeyValue{
					attribute.String(model.MetricAttributeOutcome, string(model.WorkflowRunStatusSuccess)),
				}
			case corev1.ConditionFalse:
				attrs = []attribute.KeyValue{
					attribute.String(model.MetricAttributeOutcome, string(model.WorkflowRunStatusFailure)),
				}
			}
		case relayv1beta1.RunTimedOut:
			if cond.Status == corev1.ConditionTrue {
				attrs = []attribute.KeyValue{
					attribute.String(model.MetricAttributeOutcome, string(model.WorkflowRunStatusTimedOut)),
				}
			}
		}

		if len(attrs) > 0 {
			counter := metric.Must(*r.meter).NewInt64Counter(model.MetricWorkflowRunOutcome)
			counter.Add(ctx, 1, attrs...)
		}
	}

	attrs := []attribute.KeyValue{}

	if wr.Status.CompletionTime != nil {
		totalTimeRecorder := metric.Must(*r.meter).NewInt64ValueRecorder(model.MetricWorkflowRunTotalTimeSeconds)
		totalTimeRecorder.Record(ctx, int64(wr.Status.CompletionTime.Sub(wr.CreationTimestamp.Time)/time.Second), attrs...)

		if wr.Status.StartTime != nil {
			execTimeRecorder := metric.Must(*r.meter).NewInt64ValueRecorder(model.MetricWorkflowRunExecutionTimeSeconds)
			execTimeRecorder.Record(ctx, int64(wr.Status.CompletionTime.Sub(wr.Status.StartTime.Time)/time.Second), attrs...)
		}
	}

	var initTime time.Time
	for _, step := range wr.Status.Steps {
		if step.InitializationTime != nil && (initTime.IsZero() || step.InitializationTime.Time.Before(initTime)) {
			initTime = step.InitializationTime.Time
		}
	}

	if !initTime.IsZero() {
		initTimeRecorder := metric.Must(*r.meter).NewInt64ValueRecorder(model.MetricWorkflowRunInitTimeSeconds)
		initTimeRecorder.Record(ctx, int64(initTime.Sub(wr.CreationTimestamp.Time)/time.Second), attrs...)
	}

	return ctrl.Result{}, nil
}

func Add(mgr manager.Manager, meter *metric.Meter) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&relayv1beta1.Run{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 16,
		}).
		Complete(&RunReconciler{
			client: mgr.GetClient(),
			meter:  meter,
		})
}
