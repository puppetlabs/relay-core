package trigger

import (
	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/trigger"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, o controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(o).
		For(&relayv1beta1.WebhookTrigger{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Owns(&servingv1.Service{}).
		Complete(r)
}

func Add(dm *dependency.DependencyManager) error {
	o := controller.Options{
		MaxConcurrentReconciles: dm.Config.MaxConcurrentReconciles,
	}
	return add(dm.Manager, trigger.NewReconciler(dm), o)
}
