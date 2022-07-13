package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "webhooktrigger.finalizers.controller.relay.sh"

type Reconciler struct {
	*dependency.DependencyManager

	Client client.Client
	Scheme *runtime.Scheme

	issuer authenticate.Issuer
}

func NewReconciler(dm *dependency.DependencyManager) *Reconciler {
	return &Reconciler{
		DependencyManager: dm,

		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),

		issuer: authenticate.IssuerFunc(func(ctx context.Context, claims *authenticate.Claims) (authenticate.Raw, error) {
			raw, err := authenticate.NewKeySignerIssuer(dm.JWTSigner).Issue(ctx, claims)
			if err != nil {
				return nil, err
			}

			return authenticate.NewVaultTransitWrapper(
				dm.VaultClient,
				dm.Config.VaultTransitPath,
				dm.Config.VaultTransitKey,
				authenticate.VaultTransitWrapperWithContext(authenticate.VaultTransitNamespaceContext(claims.KubernetesNamespaceUID)),
			).Wrap(ctx, raw)
		}),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	wt := obj.NewWebhookTrigger(req.NamespacedName)
	if ok, err := wt.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load WebhookTrigger")
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	deps := app.NewWebhookTriggerDeps(
		wt,
		r.issuer,
		r.Config.MetadataAPIURL,
		app.WebhookTriggerDepsWithEnvironment(r.Config.Environment),
		app.WebhookTriggerDepsWithRuntimeToolsImage(r.Config.RuntimeToolsImage),
		app.WebhookTriggerDepsWithStandaloneMode(r.Config.Standalone),
	)
	loaded, err := deps.Load(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load WebhookTrigger dependencies")
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, wt, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	// Set the ownership label first. We use this to ensure this object is
	// reconciled when the tenant changes or is deleted.
	if helper.Label(&wt.Object.ObjectMeta, model.RelayControllerTenantNameLabel, wt.Object.Spec.TenantRef.Name) {
		if err := wt.Persist(ctx, r.Client); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Delete stale dependencies regardless of upstream status. This will also
	// remove stale Knative services because they are owned by the config map.
	if _, err := deps.DeleteStale(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to delete stale WebhookTrigger dependencies")
	}

	if !loaded.Upstream {
		// Upstream dependencies (tenant, tenant dependencies) have not yet
		// settled. Wait for them to do so.
		return ctrl.Result{}, errmark.MarkTransient(fmt.Errorf("waiting for WebhookTrigger dependencies to reconcile"))
	}

	if err := app.ConfigureWebhookTriggerDeps(ctx, deps); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to configure WebhookTrigger dependencies")
	}

	if err := deps.Persist(ctx, r.Client); err != nil {
		err = errmark.MarkTransientIf(err, errhandler.RuleIsRequired)

		return ctrl.Result{}, errmap.Wrap(err, "failed to apply WebhookTrigger dependencies")
	}

	ksr := app.AsKnativeServiceResult(app.ApplyKnativeService(ctx, r.Client, deps))

	app.ConfigureWebhookTrigger(wt, ksr)

	if err := wt.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to persist WebhookTrigger status")
	}

	if !wt.Ready() {
		return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}
