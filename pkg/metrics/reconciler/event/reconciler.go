package event

import (
	"context"
	"strings"

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

type EventReconciler struct {
	client client.Client
	meter  *metric.Meter

	eventFilters map[string]*model.EventFilter
}

var _ reconcile.Reconciler = &EventReconciler{}

func (r *EventReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ev := &corev1.Event{}

	if err := r.client.Get(ctx, req.NamespacedName, ev); errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if !ev.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	r.filterMetrics(ev)

	return ctrl.Result{}, nil
}

func Add(mgr manager.Manager, meter *metric.Meter, eventFilters map[string]*model.EventFilter) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 16,
		}).
		Complete(&EventReconciler{
			client:       mgr.GetClient(),
			meter:        meter,
			eventFilters: eventFilters,
		})
}

// TODO Consider a better means of allowing customizable, dynamic event filtering
// This should include the ability to filter by object, type, reason/message, namespace, etc.
func (r *EventReconciler) filterMetrics(event *corev1.Event) {
	ef, ok := r.eventFilters[event.Type]
	if !ok {
		return
	}

	for _, filter := range ef.Filters {
		if strings.Contains(event.Reason, filter) || strings.Contains(event.Message, filter) {
			counter := metric.Must(*r.meter).NewInt64Counter(ef.Metric)
			counter.Add(context.Background(), 1,
				attribute.String(model.MetricAttributeReason, filter),
			)
		}
	}
}
