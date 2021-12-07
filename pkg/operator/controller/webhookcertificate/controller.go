package webhookcertificate

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func add(mgr manager.Manager, r reconcile.Reconciler, cfg *config.WebhookControllerConfig) error {
	return ctrl.NewControllerManagedBy(mgr)
}
