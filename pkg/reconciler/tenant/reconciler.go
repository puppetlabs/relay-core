package tenant

import (
	"context"
	"fmt"
	"time"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
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
	ReasonSucceeded = "Succeeded"
	ReasonFailed    = "Failed"
)

const (
	MessageTenantNamespaceReady = "Tenant namespace is ready"
	MessageTenantReady          = "Tenant is ready"
)

type Reconciler struct {
	*dependency.DependencyManager

	Client client.Client
	Scheme *runtime.Scheme

	metrics *controllerObservations
}

func NewReconciler(dm *dependency.DependencyManager) *Reconciler {
	return &Reconciler{
		DependencyManager: dm,

		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),

		metrics: newControllerObservations(dm.Metrics),
	}
}

func (r *Reconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
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

	if ts := tn.Object.GetDeletionTimestamp(); ts != nil && !ts.IsZero() {
		return ctrl.Result{}, nil
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

	_, err = applyNamespace(ctx, r.Client, tn)
	if err != nil {
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

func applyNamespace(ctx context.Context, cl client.Client, tenant *obj.Tenant) (*obj.Namespace, error) {
	md := tenant.Object.Spec.DeepCopy().NamespaceTemplate.Metadata

	if md.Name == "" {
		return nil, nil
	}

	ns := obj.NewNamespace(md.Name)

	found, err := ns.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	if !found {
		tenant.Own(ctx, ns)
	}

	ns.Label(ctx, model.RelayControllerTenantWorkflowLabel, "true")
	ns.LabelAnnotateFrom(ctx, md)

	if err := ns.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return ns, nil
}
