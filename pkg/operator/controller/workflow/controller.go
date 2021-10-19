package workflow

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/filter"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/handler"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/workflow"
	"github.com/puppetlabs/relay-core/pkg/util/capturer"
	tekv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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
		For(&nebulav1.WorkflowRun{}).
		Watches(&source.Kind{Type: &relayv1beta1.Tenant{}}, &handler.EnqueueRequestForReferencesByNameLabel{
			Label:      model.RelayControllerTenantNameLabel,
			TargetType: &nebulav1.WorkflowRun{},
		}).
		Watches(
			&source.Kind{Type: &tekv1beta1.PipelineRun{}},
			app.DependencyManager.NewEnqueueRequestForAnnotatedDependencyOf(&nebulav1.WorkflowRun{}),
		).
		Complete(filter.ChainR(
			r,
			errhandler.ChainReconciler(
				errhandler.WithErrorMatchers(
					errhandler.NewDefaultErrorMatchersBuilder().
						Append(
							errmark.RulePredicate(errhandler.RuleIsForbidden, func() bool { return cfg.DynamicRBACBinding }),
							errhandler.PropagatingErrorHandler,
						).
						SetFallback(capturer.CaptureErrorHandler(cfg.Capturer(), nebulav1.WorkflowRunKind)).
						Build(),
				),
				errhandler.WithPanicHandler(capturer.CapturePanicHandler(cfg.Capturer(), nebulav1.WorkflowRunKind)),
			),
			filter.ChainSingleNamespaceReconciler(cfg.Namespace),
		))
}

func Add(dm *dependency.DependencyManager) error {
	return add(dm.Manager, workflow.NewReconciler(dm), dm.Config)
}
