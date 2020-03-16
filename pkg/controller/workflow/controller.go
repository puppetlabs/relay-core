package workflow

import (
	"os"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/workflow"
	tekv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, o controller.Options) error {
	err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(o).
		For(&nebulav1.WorkflowRun{}).
		Owns(&tekv1beta1.PipelineRun{}).
		Complete(r)
	if err != nil {
		klog.Errorf("unable to create controller: %v", err)
		os.Exit(1)
	}
	return nil
}

func Add(mgr manager.Manager, dm *dependency.DependencyManager) error {
	o := controller.Options{
		MaxConcurrentReconciles: dm.Config.MaxConcurrentReconciles,
	}
	return add(mgr, workflow.NewReconciler(mgr, dm), o)
}
