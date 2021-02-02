package reconciler

import (
	"context"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/metrics/model"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
	"go.opentelemetry.io/otel/label"
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

func (r *WorkflowRunReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

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
		counter := metric.Must(*r.meter).NewInt64Counter(model.MetricWorkflowRunOutcome)
		counter.Add(context.Background(), 1,
			label.String(model.MetricLabelOutcome, status),
		)
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
