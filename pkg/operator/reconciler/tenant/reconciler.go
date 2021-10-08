package tenant

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
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

func NewReconciler(cl client.Client, cfg *config.WorkflowControllerConfig) *Reconciler {
	return &Reconciler{
		Client: cl,
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

	deps := app.NewTenantDeps(tn, app.TenantDepsWithStandaloneMode(r.Config.Standalone))
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

	err = app.ConfigureTenantDeps(ctx, deps)
	if err != nil {
		return ctrl.Result{}, err
	}

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
