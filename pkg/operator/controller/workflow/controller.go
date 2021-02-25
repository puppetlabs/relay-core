package workflow

import (
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/filter"
	"github.com/puppetlabs/relay-core/pkg/operator/reconciler/workflow"
	tekv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config.WorkflowControllerConfig) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: cfg.MaxConcurrentReconciles,
		}).
		For(&nebulav1.WorkflowRun{}).
		Owns(&tekv1beta1.PipelineRun{}).
		Complete(filter.ChainRight(r,
			filter.ErrorCaptureReconcilerLink(
				&nebulav1.WorkflowRun{},
				cfg.Capturer(),
				filter.ErrorCaptureReconcilerWithAdditionalTransientRule(
					errmark.RulePredicate(errmark.TransientIfForbidden, func() bool { return cfg.DynamicRBACBinding }),
				),
			),
			filter.NamespaceFilterReconcilerLink(cfg.Namespace),
		))
}

func Add(dm *dependency.DependencyManager) error {
	return add(dm.Manager, workflow.NewReconciler(dm), dm.Config)
}
