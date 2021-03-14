package tenant

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	pvpoolv1alpha1obj "github.com/puppetlabs/pvpool/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "tenant.finalizers.controller.relay.sh"

type Reconciler struct {
	Client client.Client
	Config *config.WorkflowControllerConfig
}

func NewReconciler(client client.Client, cfg *config.WorkflowControllerConfig) *Reconciler {
	return &Reconciler{
		Client: client,
		Config: cfg,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	tn := obj.NewTenant(req.NamespacedName)
	if ok, err := tn.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	opts := []app.TenantDepsOption{app.TenantDepsWithStandaloneMode(r.Config.Standalone)}
	if p := r.Config.ToolInjectionPool; p != nil {
		opts = append(opts, app.TenantDepsWithToolInjectionPool(pvpoolv1alpha1obj.NewPool(*p)))
	}

	deps := app.NewTenantDeps(tn, opts...)
	if _, err := deps.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, tn, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if _, err := deps.DeleteStale(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to delete stale dependencies")
	}

	app.ConfigureTenantDeps(ctx, deps)

	tdr := app.AsTenantDepsResult(deps, deps.Persist(ctx, r.Client))

	if tdr.Error != nil {
		return ctrl.Result{}, tdr.Error
	}

	app.ConfigureTenant(tn, tdr)

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
