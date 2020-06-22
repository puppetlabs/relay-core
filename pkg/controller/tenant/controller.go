package tenant

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/config"
	"github.com/puppetlabs/relay-core/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/reconciler/filter"
	"github.com/puppetlabs/relay-core/pkg/reconciler/tenant"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config.WorkflowControllerConfig) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: cfg.MaxConcurrentReconciles,
		}).
		For(&relayv1beta1.Tenant{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(filter.ChainRight(r,
			filter.ErrorCaptureReconcilerLink(
				&relayv1beta1.Tenant{},
				cfg.Capturer(),
				filter.ErrorCaptureReconcilerWithAdditionalTransientRule(
					errmark.TransientPredicate(errmark.TransientIfForbidden, func() bool { return cfg.DynamicRBACBinding }),
				),
			),
			filter.NamespaceFilterReconcilerLink(cfg.Namespace),
		))
}

func Add(mgr manager.Manager, cfg *config.WorkflowControllerConfig) error {
	return add(mgr, tenant.NewReconciler(mgr.GetClient(), cfg), cfg)
}
