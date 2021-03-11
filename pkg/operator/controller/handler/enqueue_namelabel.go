package handler

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	EnqueueRequestForReferencesByNameLabelTimeout = 30 * time.Second
)

type EnqueueRequestForReferencesByNameLabel struct {
	Label      string
	TargetType runtime.Object
	gvk        schema.GroupVersionKind
	cl         client.Client
}

var _ handler.EventHandler = &EnqueueRequestForReferencesByNameLabel{}
var _ inject.Client = &EnqueueRequestForReferencesByNameLabel{}
var _ inject.Scheme = &EnqueueRequestForReferencesByNameLabel{}

func (e *EnqueueRequestForReferencesByNameLabel) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

func (e *EnqueueRequestForReferencesByNameLabel) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.ObjectOld, q)
	e.add(evt.ObjectNew, q)
}

func (e *EnqueueRequestForReferencesByNameLabel) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

func (e *EnqueueRequestForReferencesByNameLabel) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

func (e *EnqueueRequestForReferencesByNameLabel) add(target client.Object, q workqueue.RateLimitingInterface) {
	ctx, cancel := context.WithTimeout(context.Background(), EnqueueRequestForReferencesByNameLabelTimeout)
	defer cancel()

	var objs unstructured.UnstructuredList
	objs.SetGroupVersionKind(e.gvk)

	if err := e.cl.List(ctx, &objs, client.InNamespace(target.GetNamespace()), client.MatchingLabels{e.Label: target.GetName()}); err != nil {
		klog.Errorf("enqueue: failed to list resources referencing %s %s/%s by label: %+v", e.gvk, target.GetNamespace(), target.GetName(), err)

		// No choice but to panic here. Missing this reconcile could mean that
		// downstream dependencies don't get updated, so we'll try to restart
		// the whole process.
		panic(err)
	}

	for _, obj := range objs.Items {
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
		}
		q.Add(req)
		klog.V(4).Infof("enqueue: successful enqueue of %s %s", e.gvk, req.NamespacedName)
	}
}

func (e *EnqueueRequestForReferencesByNameLabel) InjectClient(cl client.Client) error {
	e.cl = cl
	return nil
}

func (e *EnqueueRequestForReferencesByNameLabel) InjectScheme(s *runtime.Scheme) error {
	kinds, _, err := s.ObjectKinds(e.TargetType)
	if err != nil {
		return err
	}

	e.gvk = kinds[0]
	return nil
}
