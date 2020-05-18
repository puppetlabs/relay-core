package filter

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type NamespaceFilterReconciler struct {
	name     string
	delegate reconcile.Reconciler
}

var _ reconcile.Reconciler = &NamespaceFilterReconciler{}

func (nfr NamespaceFilterReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	// You can't be clever and use the built-in namespace restrictions or
	// predicates in controller-runtime to filter out the namespace before it
	// gets here. The caching applies to the same namespace filter, so the
	// namespaces used/created by this controller will appear to not exist!
	if nfr.name != "" && req.Namespace != nfr.name {
		return ctrl.Result{}, nil
	}

	return nfr.delegate.Reconcile(req)
}

func NewNamespaceFilterReconciler(namespace string, delegate reconcile.Reconciler) *NamespaceFilterReconciler {
	return &NamespaceFilterReconciler{
		name:     namespace,
		delegate: delegate,
	}
}
