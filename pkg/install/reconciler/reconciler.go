package reconciler

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/relay-core/pkg/install/app"
	"github.com/puppetlabs/relay-core/pkg/install/dependency"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func New(dm *dependency.Manager) *Reconciler {
	return &Reconciler{
		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	core := obj.NewCore(req.NamespacedName)
	if ok, err := core.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	} else if !ok {
		return ctrl.Result{}, nil
	}

	var cd *app.CoreDeps
	cd, err := app.ApplyCoreDeps(
		ctx,
		r.Client,
		core,
	)
	if err != nil {
		err = errmark.MarkTransientIf(err, errhandler.RuleIsRequired)

		return ctrl.Result{}, errmap.Wrap(err, "failed to apply dependencies")
	}

	app.ConfigureCore(ctx, cd)

	if err := core.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to persist Run")
	}

	return ctrl.Result{}, nil
}
