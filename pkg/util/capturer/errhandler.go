package capturer

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func CaptureErrorHandler(capturer trackers.Capturer, gvk schema.GroupVersionKind) errhandler.ErrorHandler {
	return errhandler.ErrorHandlerFunc(func(ctx context.Context, req reconcile.Request, err error) (reconcile.Result, error) {
		cap := capturer.WithTags(
			trackers.Tag{Key: "k8s.api-version", Value: gvk.GroupVersion().String()},
			trackers.Tag{Key: "k8s.kind", Value: gvk.Kind},
			trackers.Tag{Key: "k8s.metadata.namespace", Value: req.Namespace},
			trackers.Tag{Key: "k8s.metadata.name", Value: req.Name},
		)
		cap.Capture(err).Report(ctx)
		return reconcile.Result{}, err
	})
}

func CapturePanicHandler(capturer trackers.Capturer, gvk schema.GroupVersionKind) errhandler.PanicHandler {
	return errhandler.PanicHandlerFunc(func(ctx context.Context, req reconcile.Request, rv interface{}) (reconcile.Result, error) {
		cap := capturer.WithTags(
			trackers.Tag{Key: "k8s.api-version", Value: gvk.GroupVersion().String()},
			trackers.Tag{Key: "k8s.kind", Value: gvk.Kind},
			trackers.Tag{Key: "k8s.metadata.namespace", Value: req.Namespace},
			trackers.Tag{Key: "k8s.metadata.name", Value: req.Name},
		)
		switch rvt := rv.(type) {
		case error:
			cap.Capture(rvt).Report(ctx)
		default:
			cap.CaptureMessage(fmt.Sprintf("%+v", rvt)).Report(ctx)
		}
		panic(rv)
	})
}
