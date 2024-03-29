package reconciler

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/install/app"
	"github.com/puppetlabs/relay-core/pkg/install/dependency"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "installer.finalizers.controller.relay.sh"

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
	klog.Infof("reconciling request: %v", req)
	core := obj.NewCore(req.NamespacedName)
	if ok, err := core.Load(ctx, r.Client); err != nil {
		klog.Error(err)
		return ctrl.Result{}, errmap.Wrap(err, "failed to load RelayCore")
	} else if !ok {
		return ctrl.Result{}, nil
	}

	cd := app.NewCoreDeps(core)

	if _, err := cd.Load(ctx, r.Client); err != nil {
		klog.Error(err)
		return ctrl.Result{}, errmap.Wrap(err, "failed to load RelayCore dependencies")
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, core, func() error {
		_, err := cd.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	_, err = app.ApplyCoreDeps(
		ctx,
		r.Client,
		core,
	)
	if err != nil {
		klog.Error(err)
		err = errmark.MarkTransientIf(err, errhandler.RuleIsRequired)

		return ctrl.Result{}, errmap.Wrap(err, "failed to apply RelayCore dependencies")
	}

	if err := core.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to persist RelayCore status")
	}

	return ctrl.Result{}, nil
}
