package controller

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/filter"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/install/config"
	"github.com/puppetlabs/relay-core/pkg/install/dependency"
	"github.com/puppetlabs/relay-core/pkg/install/reconciler"
	"github.com/puppetlabs/relay-core/pkg/util/capturer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config.InstallerControllerConfig) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: cfg.MaxConcurrentReconciles,
		}).
		For(&v1alpha1.RelayCore{}).
		Complete(filter.ChainR(
			r, errhandler.ChainReconciler(
				errhandler.WithErrorMatchers(
					errhandler.NewDefaultErrorMatchersBuilder().
						SetFallback(capturer.CaptureErrorHandler(cfg.Capturer(), v1alpha1.RelayCoreKind)).
						Build(),
				),
				errhandler.WithPanicHandler(capturer.CapturePanicHandler(cfg.Capturer(), v1alpha1.RelayCoreKind)),
			),
			filter.ChainSingleNamespaceReconciler(cfg.Namespace),
		))
}

func Add(dm *dependency.Manager) error {
	return add(dm.Manager, reconciler.New(dm), dm.Config)
}
