package app

import (
	"context"

	pvpoolv1alpha1obj "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PoolRefPredicatedCheckout is an instance of Checkout that only persists when
// the checkout has a pool set.
type PoolRefPredicatedCheckout struct {
	*pvpoolv1alpha1obj.Checkout
}

func (prpc *PoolRefPredicatedCheckout) Persist(ctx context.Context, cl client.Client) error {
	if !prpc.Satisfied() {
		return nil
	}

	return prpc.Checkout.Persist(ctx, cl)
}

func (prpc *PoolRefPredicatedCheckout) Satisfied() bool {
	return prpc.Object.Spec.PoolRef.Name != ""
}
