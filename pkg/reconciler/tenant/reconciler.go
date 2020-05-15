package tenant

import (
	"context"
	"fmt"
	"time"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FinalizerName = "tenant.finalizers.controller.relay.sh"

	ReasonSucceeded = "Succeeded"
	ReasonFailed    = "Failed"

	MessageTenantNamespaceReady = "Tenant namespace is ready"
	MessageTenantReady          = "Tenant is ready"
)

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

func (r *Reconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	// You can't be clever and use the built-in namespace restrictions or
	// predicates in controller-runtime to filter out the namespace before it
	// gets here. The caching applies to the same namespace filter, so the
	// namespaces used/created by this controller will appear to not exist!
	if r.Config.Namespace != "" && req.Namespace != r.Config.Namespace {
		return ctrl.Result{}, nil
	}

	klog.Infof("reconciling Tenant %s", req.NamespacedName)
	defer func() {
		if err != nil {
			klog.Infof("error reconciling Tenant %s: %+v", req.NamespacedName, err)
		} else {
			klog.Infof("done reconciling Tenant %s", req.NamespacedName)
		}
	}()

	ctx := context.Background()

	tn := obj.NewTenant(req.NamespacedName)
	if ok, err := tn.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to load dependencies: %+v", err)
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	deps := obj.NewTenantDeps(tn)
	if _, err := deps.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to load dependencies: %+v", err)
	}

	finalized, err := obj.Finalize(ctx, r.Client, FinalizerName, tn, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	tn.Object.Status.ObservedGeneration = tn.Object.Generation

	tnsc := relayv1beta1.TenantCondition{
		Condition: relayv1beta1.Condition{
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonSucceeded,
			Message:            MessageTenantNamespaceReady,
		},
		Type: relayv1beta1.TenantNamespaceReady,
	}

	tnc := relayv1beta1.TenantCondition{
		Condition: relayv1beta1.Condition{
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonSucceeded,
			Message:            MessageTenantReady,
		},
		Type: relayv1beta1.TenantReady,
	}

	obj.ConfigureTenantDeps(ctx, deps)

	if err := deps.Persist(ctx, r.Client); err != nil {
		tnsc.Condition = relayv1beta1.Condition{
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             ReasonFailed,
			Message:            err.Error(),
		}

		if err := tn.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	tn.Object.Status.Conditions = []relayv1beta1.TenantCondition{tnsc, tnc}

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
