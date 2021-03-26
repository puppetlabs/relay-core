package workflow

import (
	"context"
	"time"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/metrics/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type WorkflowRunReconciler struct {
	client client.Client
	meter  *metric.Meter
}

var _ reconcile.Reconciler = &WorkflowRunReconciler{}

func (r *WorkflowRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	wr := &nebulav1.WorkflowRun{}

	if err := r.client.Get(ctx, req.NamespacedName, wr); errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if !wr.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	status := wr.Status.Status
	switch status {
	case string(obj.WorkflowRunStatusSuccess), string(obj.WorkflowRunStatusFailure),
		string(obj.WorkflowRunStatusCancelled), string(obj.WorkflowRunStatusTimedOut), string(obj.WorkflowRunStatusSkipped):
		attrs := []attribute.KeyValue{
			attribute.String(model.MetricAttributeOutcome, status),
		}

		counter := metric.Must(*r.meter).NewInt64Counter(model.MetricWorkflowRunOutcome)
		counter.Add(ctx, 1, attrs...)

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
			if step.InitTime != nil && (initTime.IsZero() || step.InitTime.Time.Before(initTime)) {
				initTime = step.InitTime.Time
			}
		}

		if !initTime.IsZero() {
			initTimeRecorder := metric.Must(*r.meter).NewInt64ValueRecorder(model.MetricWorkflowRunInitTimeSeconds)
			initTimeRecorder.Record(ctx, int64(initTime.Sub(wr.CreationTimestamp.Time)/time.Second), attrs...)
		}
	}

	return ctrl.Result{}, nil
}

func Add(mgr manager.Manager, meter *metric.Meter) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nebulav1.WorkflowRun{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 16,
		}).
		Complete(&WorkflowRunReconciler{
			client: mgr.GetClient(),
			meter:  meter,
		})
}
