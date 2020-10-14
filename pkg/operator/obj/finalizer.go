package obj

import (
	"context"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FinalizablePersister interface {
	Finalizable
	Persister
}

func Finalize(ctx context.Context, cl client.Client, name string, obj FinalizablePersister, run func() error) (bool, error) {
	if obj.Finalizing() {
		klog.Infof("running finalizer %s", name)

		if err := run(); err != nil {
			return false, err
		}

		obj.RemoveFinalizer(ctx, name)

		err := obj.Persist(ctx, cl)
		return err == nil, err
	} else if obj.AddFinalizer(ctx, name) {
		return false, obj.Persist(ctx, cl)
	}

	return false, nil
}
