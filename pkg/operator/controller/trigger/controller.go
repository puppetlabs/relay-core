package trigger

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/filter"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/handler"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/trigger"
	"github.com/puppetlabs/relay-core/pkg/util/capturer"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config.WorkflowControllerConfig) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: cfg.MaxConcurrentReconciles,
		}).
		For(&relayv1beta1.WebhookTrigger{}).
		Watches(&source.Kind{Type: &relayv1beta1.Tenant{}}, &handler.EnqueueRequestForReferencesByNameLabel{
			Label:      model.RelayControllerTenantNameLabel,
			TargetType: &relayv1beta1.WebhookTrigger{},
		}).
		Watches(&source.Kind{Type: &servingv1.Service{}}, &handler.EnqueueRequestForAnnotatedDependent{OwnerType: &relayv1beta1.WebhookTrigger{}}).
		Complete(filter.ChainR(
			r,
			errhandler.ChainReconciler(
				errhandler.WithErrorMatchers(
					errhandler.NewDefaultErrorMatchersBuilder().
						Append(
							errmark.RulePredicate(errhandler.RuleIsForbidden, func() bool { return cfg.DynamicRBACBinding }),
							errhandler.PropagatingErrorHandler,
						).
						SetFallback(capturer.CaptureErrorHandler(cfg.Capturer(), relayv1beta1.WebhookTriggerKind)).
						Build(),
				),
				errhandler.WithPanicHandler(capturer.CaptureErrorHandler(cfg.Capturer(), relayv1beta1.WebhookTriggerKind)),
			),
			filter.ChainSingleNamespaceReconciler(cfg.Namespace),
		))
}

func Add(dm *dependency.DependencyManager) error {
	return add(dm.Manager, trigger.NewReconciler(dm), dm.Config)
}
