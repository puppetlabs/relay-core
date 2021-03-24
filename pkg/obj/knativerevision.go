package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	KnativeRevisionKind = servingv1.SchemeGroupVersion.WithKind("Revision")
)

type KnativeRevision struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *servingv1.Revision
}

func makeKnativeRevision(key client.ObjectKey, obj *servingv1.Revision) *KnativeRevision {
	kr := &KnativeRevision{Key: key, Object: obj}
	kr.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&kr.Key, lifecycle.TypedObject{GVK: KnativeRevisionKind, Object: kr.Object})
	return kr
}

func (kr *KnativeRevision) Copy() *KnativeRevision {
	return makeKnativeRevision(kr.Key, kr.Object.DeepCopy())
}

func NewKnativeRevision(key client.ObjectKey) *KnativeRevision {
	return makeKnativeRevision(key, &servingv1.Revision{})
}

func NewKnativeRevisionFromObject(obj *servingv1.Revision) *KnativeRevision {
	return makeKnativeRevision(client.ObjectKeyFromObject(obj), obj)
}

func NewKnativeRevisionPatcher(upd, orig *KnativeRevision) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
