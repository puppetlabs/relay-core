package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ConditionKind = tektonv1alpha1.SchemeGroupVersion.WithKind("Condition")
)

type Condition struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *tektonv1alpha1.Condition
}

func makeCondition(key client.ObjectKey, obj *tektonv1alpha1.Condition) *Condition {
	c := &Condition{Key: key, Object: obj}
	c.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&c.Key, lifecycle.TypedObject{GVK: ConditionKind, Object: c.Object})
	return c
}

func (c *Condition) Copy() *Condition {
	return makeCondition(c.Key, c.Object.DeepCopy())
}

func NewCondition(key client.ObjectKey) *Condition {
	return makeCondition(key, &tektonv1alpha1.Condition{})
}

func NewConditionFromObject(obj *tektonv1alpha1.Condition) *Condition {
	return makeCondition(client.ObjectKeyFromObject(obj), obj)
}

func NewConditionPatcher(upd, orig *Condition) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
