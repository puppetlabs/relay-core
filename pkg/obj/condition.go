package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Condition struct {
	Key    client.ObjectKey
	Object *tektonv1alpha1.Condition
}

var _ lifecycle.LabelAnnotatableFrom = &Condition{}
var _ lifecycle.Loader = &Condition{}
var _ lifecycle.Ownable = &Condition{}
var _ lifecycle.Persister = &Condition{}

func (c *Condition) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&c.Object.ObjectMeta, from)
}

func (c *Condition) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(c.Object, owner)
}

func (c *Condition) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, c.Key, c.Object)
}

func (c *Condition) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, c.Object, helper.WithObjectKey(c.Key))
}

func NewCondition(key client.ObjectKey) *Condition {
	return &Condition{
		Key:    key,
		Object: &tektonv1alpha1.Condition{},
	}
}
