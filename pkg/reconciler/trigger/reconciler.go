package trigger

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
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
	klog.Infof("reconciling WebhookTrigger %s", req.NamespacedName)
	defer func() {
		if err != nil {
			klog.Infof("error reconciling WebhookTrigger %s: %+v", req.NamespacedName, err)
		} else {
			klog.Infof("done reconciling WebhookTrigger %s", req.NamespacedName)
		}
	}()

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
		return ctrl.Result{}, fmt.Errorf("failed to load dependencies: %+v", err)
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
		return ctrl.Result{}, fmt.Errorf("failed to apply dependencies: %+v", err)
	}

	ksr := obj.AsKnativeServiceResult(obj.ApplyKnativeService(ctx, r.Client, deps))

	obj.ConfigureWebhookTrigger(wt, ksr)

	if err := wt.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
