package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	pvpoolv1alpha1obj "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1/obj"
	"github.com/puppetlabs/relay-core/pkg/obj"
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

func ConfigureToolInjectionCheckout(co *PoolRefPredicatedCheckout, t *obj.Tenant, pr pvpoolv1alpha1.PoolReference) {
	// If we already have a volume, this checkout is completely configured.
	if co.Object.Status.VolumeName != "" {
		return
	}

	// No pool reference means this controller isn't set up to do tool injection.
	if pr.Name == "" {
		return
	}

	// No volume claim template means we don't want tool injection.
	vct := t.Object.Spec.ToolInjection.VolumeClaimTemplate
	if vct == nil {
		return
	}

	co.Object.Spec = pvpoolv1alpha1.CheckoutSpec{
		PoolRef:     pr,
		ClaimName:   co.Key.Name,
		AccessModes: vct.Spec.AccessModes,
	}
}

// TODO: This method should be able to go away or be merged with
// ConfigureToolInjectionCheckout once we can read the tenant from a Run.
func ConfigureToolInjectionCheckoutForWorkflowRun(co *PoolRefPredicatedCheckout, wr *obj.WorkflowRun, t *obj.Tenant, pr pvpoolv1alpha1.PoolReference) {
	// If we already have a volume, this checkout is completely configured.
	if co.Object.Status.VolumeName != "" {
		return
	}

	// No pool reference means this controller isn't set up to do tool injection.
	if pr.Name == "" {
		return
	}

	// No volume claim template means we don't want tool injection.
	vct := t.Object.Spec.ToolInjection.VolumeClaimTemplate
	if vct == nil {
		return
	}

	co.Object.Spec = pvpoolv1alpha1.CheckoutSpec{
		PoolRef:     pr,
		ClaimName:   co.Key.Name,
		AccessModes: vct.Spec.AccessModes,
	}
}

// CheckoutSet is a collection of checkouts identified by common list options.
type CheckoutSet struct {
	ListOptions *client.ListOptions

	Checkouts []*pvpoolv1alpha1obj.Checkout
}

var _ lifecycle.Loader = &CheckoutSet{}

func (cs *CheckoutSet) Load(ctx context.Context, cl client.Client) (bool, error) {
	checkouts := &pvpoolv1alpha1.CheckoutList{}
	if err := cl.List(ctx, checkouts, cs.ListOptions); err != nil {
		return false, err
	}

	cs.Checkouts = make([]*pvpoolv1alpha1obj.Checkout, len(checkouts.Items))
	for i := range checkouts.Items {
		cs.Checkouts[i] = pvpoolv1alpha1obj.NewCheckoutFromObject(&checkouts.Items[i])
	}

	return true, nil
}

func NewCheckoutSet(opts ...client.ListOption) *CheckoutSet {
	o := &client.ListOptions{}
	o.ApplyOptions(opts)

	return &CheckoutSet{
		ListOptions: o,
	}
}

func RemoveCheckoutsWithClaimNames(initial []*pvpoolv1alpha1obj.Checkout, claimNames []string) (final []*pvpoolv1alpha1obj.Checkout) {
	checkoutsByClaimName := make(map[string]*pvpoolv1alpha1obj.Checkout)
	for _, checkout := range initial {
		claimName := checkout.Object.Status.VolumeClaimRef.Name
		if claimName == "" {
			claimName = checkout.Object.Spec.ClaimName
		}
		if claimName == "" {
			final = append(final, checkout)
			continue
		}

		checkoutsByClaimName[claimName] = checkout
	}

	for _, claimName := range claimNames {
		delete(checkoutsByClaimName, claimName)
	}

	for _, checkout := range checkoutsByClaimName {
		final = append(final, checkout)
	}
	return
}
