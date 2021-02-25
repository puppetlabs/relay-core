package tenant

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/filter"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/tenant"
	"github.com/puppetlabs/relay-core/pkg/util/capturer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		Complete(filter.ChainR(
			r,
			errhandler.ChainReconciler(
				errhandler.WithErrorMatchers(
					errhandler.NewDefaultErrorMatchersBuilder().
						Append(
							errmark.RulePredicate(errhandler.RuleIsForbidden, func() bool { return cfg.DynamicRBACBinding }),
							errhandler.PropagatingErrorHandler,
						).
						SetFallback(capturer.CaptureErrorHandler(cfg.Capturer(), relayv1beta1.TenantKind)).
						Build(),
				),
				errhandler.WithPanicHandler(capturer.CaptureErrorHandler(cfg.Capturer(), relayv1beta1.TenantKind)),
			),
			filter.ChainSingleNamespaceReconciler(cfg.Namespace),
		))
}

func Add(mgr manager.Manager, cfg *config.WorkflowControllerConfig) error {
	mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.PersistentVolume{}, "status.phase", func(o runtime.Object) []string {
		var res []string
		vol, ok := o.(*corev1.PersistentVolume)
		if ok {
			res = append(res, string(vol.Status.Phase))
		}
		return res
	})

	return add(mgr, tenant.NewReconciler(mgr.GetClient(), cfg), cfg)
}
