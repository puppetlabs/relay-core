package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/dependency"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/reconciler/errmark"
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

	metrics *controllerObservations
	issuer  authenticate.Issuer
}

func NewReconciler(dm *dependency.DependencyManager) *Reconciler {
	return &Reconciler{
		DependencyManager: dm,

		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),

		metrics: newControllerObservations(dm.Metrics),

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

func (r *Reconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()

	wt := obj.NewWebhookTrigger(req.NamespacedName)
	if ok, err := wt.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to load dependencies: %+v", err)
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	deps := obj.NewWebhookTriggerDeps(wt, r.issuer, r.Config.MetadataAPIURL)
	if _, err := deps.Load(ctx, r.Client); err != nil {
		// Could be waiting for the tenant to reconcile.
		err = errmark.MarkTransient(err, errmark.TransientIfObjectRequired).
			Map(func(err error) error {
				return fmt.Errorf("failed to load dependencies: %+v", err)
			}).
			Resolve()
		return ctrl.Result{}, err
	}

	finalized, err := obj.Finalize(ctx, r.Client, FinalizerName, wt, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if err := obj.ConfigureWebhookTriggerDeps(ctx, deps); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to configure dependencies: %+v", err)
	}

	if err := deps.Persist(ctx, r.Client); err != nil {
		err = errmark.MarkTransient(err, errmark.TransientIfObjectRequired).
			Map(func(err error) error {
				return fmt.Errorf("failed to apply dependencies: %+v", err)
			}).
			Resolve()
		return ctrl.Result{}, err
	}

	ksr := obj.AsKnativeServiceResult(obj.ApplyKnativeService(ctx, r.Client, deps))

	obj.ConfigureWebhookTrigger(wt, ksr)

	if err := wt.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if !wt.Ready() {
		return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}
