package filter

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"github.com/puppetlabs/relay-core/pkg/errmark"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

type ErrorCaptureReconciler struct {
	apiType        runtime.Object
	capturer       trackers.Capturer
	transientRules []errmark.TransientRule
	delegate       reconcile.Reconciler

	gvk schema.GroupVersionKind
}

var _ reconcile.Reconciler = &ErrorCaptureReconciler{}
var _ inject.Scheme = &ErrorCaptureReconciler{}
var _ inject.Injector = &ErrorCaptureReconciler{}

func (ecr ErrorCaptureReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	capturer := ecr.capturer.WithTags(
		trackers.Tag{Key: "k8s.api-version", Value: ecr.gvk.GroupVersion().String()},
		trackers.Tag{Key: "k8s.kind", Value: ecr.gvk.Kind},
		trackers.Tag{Key: "k8s.metadata.namespace", Value: req.Namespace},
		trackers.Tag{Key: "k8s.metadata.name", Value: req.Name},
	)

	klog.Infof("reconciling %s %s", ecr.gvk.Kind, req.NamespacedName)
	perr := capturer.Try(context.Background(), func(ctx context.Context) {
		result, err = ecr.delegate.Reconcile(req)
		if err != nil {
			err = errmark.Resolve(errmark.MarkTransient(err, ecr.transientRules...))

			errmark.IfAll(err, []errmark.IfFunc{errmark.IfNotTransient, errmark.IfNotUser}, func(err error) {
				capturer.Capture(err).Report(ctx)
			})

			klog.Infof("error reconciling %s %s: %+v", ecr.gvk.Kind, req.NamespacedName, err)
		} else {
			klog.Infof("done reconciling %s %s", ecr.gvk.Kind, req.NamespacedName)
		}
	})
	if perr != nil {
		panic(perr)
	}

	return
}

func (ecr *ErrorCaptureReconciler) InjectScheme(scheme *runtime.Scheme) (err error) {
	ecr.gvk, err = apiutil.GVKForObject(ecr.apiType, scheme)
	return
}

func (ecr *ErrorCaptureReconciler) InjectFunc(f inject.Func) error {
	return f(ecr.delegate)
}

type ErrorCaptureReconcilerOption func(ecr *ErrorCaptureReconciler)

func ErrorCaptureReconcilerWithAdditionalTransientRule(rule errmark.TransientRule) ErrorCaptureReconcilerOption {
	return func(ecr *ErrorCaptureReconciler) {
		ecr.transientRules = append(ecr.transientRules, rule)
	}
}

func NewErrorCaptureReconciler(apiType runtime.Object, capturer trackers.Capturer, delegate reconcile.Reconciler, opts ...ErrorCaptureReconcilerOption) *ErrorCaptureReconciler {
	ecr := &ErrorCaptureReconciler{
		apiType:        apiType,
		capturer:       capturer,
		transientRules: []errmark.TransientRule{errmark.TransientDefault},
		delegate:       delegate,
	}

	for _, opt := range opts {
		opt(ecr)
	}

	return ecr
}

func ErrorCaptureReconcilerLink(apiType runtime.Object, capturer trackers.Capturer, opts ...ErrorCaptureReconcilerOption) ChainLink {
	return func(delegate reconcile.Reconciler) reconcile.Reconciler {
		return NewErrorCaptureReconciler(apiType, capturer, delegate, opts...)
	}
}
