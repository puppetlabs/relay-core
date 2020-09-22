package filter

import "sigs.k8s.io/controller-runtime/pkg/reconcile"

type ChainLink func(reconcile.Reconciler) reconcile.Reconciler

func ChainRight(last reconcile.Reconciler, links ...ChainLink) reconcile.Reconciler {
	for i := len(links) - 1; i >= 0; i-- {
		last = links[i](last)
	}

	return last
}
