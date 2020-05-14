package trigger

import (
	"context"
	"fmt"
	"time"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReasonWebhookTriggerReady           = "WebhookTriggerReady"
	ReasonWebhookTriggerNotReady        = "WebhookTriggerNotReady"
	ReasonWebhookTriggerServiceReady    = "WebhookTriggerServiceReady"
	ReasonWebhookTriggerServiceNotReady = "WebhookTriggerServiceNotReady"
	ReasonWebhookTriggerServiceError    = "WebhookTriggerServiceError"
)

const (
	MessageWebhookTriggerServiceReady    = "Webhook trigger service is ready"
	MessageWebhookTriggerServiceNotReady = "Webhook trigger service is not ready"
	MessageWebhookTriggerReady           = "Webhook trigger is ready"
	MessageWebhookTriggerNotReady        = "Webhook trigger is not ready"
)

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

	if ts := wt.Object.GetDeletionTimestamp(); ts != nil && !ts.IsZero() {
		return ctrl.Result{}, nil
	}

	wtd, err := obj.ApplyTriggerDeps(
		ctx,
		r.Client,
		wt,
		r.issuer,
		r.Config.MetadataAPIURL,
		obj.WebhookTriggerDepsWithSourceSystemImagePullSecret(r.Config.ImagePullSecretKey()),
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply dependencies: %+v", err)
	}

	wt.Object.Status.ObservedGeneration = wt.Object.Generation

	wtc := relayv1beta1.WebhookTriggerCondition{
		Condition: relayv1beta1.Condition{
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonWebhookTriggerReady,
			Message:            MessageWebhookTriggerReady,
		},
		Type: relayv1beta1.WebhookTriggerReady,
	}

	wtsc := relayv1beta1.WebhookTriggerCondition{
		Condition: relayv1beta1.Condition{
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonWebhookTriggerServiceReady,
			Message:            MessageWebhookTriggerServiceReady,
		},
		Type: relayv1beta1.WebhookTriggerServiceReady,
	}

	kns, err := obj.ApplyKnativeService(ctx, r.Client, wtd)
	if err != nil {
		wtsc.Condition = relayv1beta1.Condition{
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonWebhookTriggerServiceError,
			Message:            err.Error(),
		}

		if err := wt.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	for _, condition := range kns.Object.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			wtsc.Condition = relayv1beta1.Condition{
				Status:             corev1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             ReasonWebhookTriggerServiceNotReady,
				Message:            MessageWebhookTriggerServiceNotReady,
			}
		}
	}

	url := kns.Object.Status.URL
	if url != nil {
		wt.Object.Status.URL = url.String()
	} else {
		wtc.Condition = relayv1beta1.Condition{
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonWebhookTriggerNotReady,
			Message:            MessageWebhookTriggerNotReady,
		}
	}

	wt.Object.Status.Conditions = []relayv1beta1.WebhookTriggerCondition{wtc, wtsc}

	if err := wt.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
