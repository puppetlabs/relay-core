package trigger

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/handler"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/filter"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/trigger"
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
		Complete(filter.ChainRight(r,
			filter.ErrorCaptureReconcilerLink(
				&relayv1beta1.WebhookTrigger{},
				cfg.Capturer(),
				filter.ErrorCaptureReconcilerWithAdditionalTransientRule(
					errmark.RulePredicate(errmark.TransientIfForbidden, func() bool { return cfg.DynamicRBACBinding }),
				),
			),
			filter.NamespaceFilterReconcilerLink(cfg.Namespace),
		))
}

func Add(dm *dependency.DependencyManager) error {
	return add(dm.Manager, trigger.NewReconciler(dm), dm.Config)
}
