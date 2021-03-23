package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	KnativeServiceKind = servingv1.SchemeGroupVersion.WithKind("Service")
)

type KnativeService struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *servingv1.Service
}

func makeKnativeService(key client.ObjectKey, obj *servingv1.Service) *KnativeService {
	ks := &KnativeService{Key: key, Object: obj}
	ks.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&ks.Key, lifecycle.TypedObject{GVK: KnativeServiceKind, Object: ks.Object})
	return ks
}

func (ks *KnativeService) Copy() *KnativeService {
	return makeKnativeService(ks.Key, ks.Object.DeepCopy())
}

func NewKnativeService(key client.ObjectKey) *KnativeService {
	return makeKnativeService(key, &servingv1.Service{})
}

func NewKnativeServiceFromObject(obj *servingv1.Service) *KnativeService {
	return makeKnativeService(client.ObjectKeyFromObject(obj), obj)
}

func NewKnativeServicePatcher(upd, orig *KnativeService) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
