package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Task struct {
	Key    client.ObjectKey
	Object *tektonv1beta1.Task
}

var _ lifecycle.LabelAnnotatableFrom = &Task{}
var _ lifecycle.Loader = &Task{}
var _ lifecycle.Ownable = &Task{}
var _ lifecycle.Persister = &Task{}

func (t *Task) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&t.Object.ObjectMeta, from)
}

func (t *Task) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, t.Key, t.Object)
}

func (t *Task) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(t.Object, owner)
}

func (t *Task) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, t.Object, helper.WithObjectKey(t.Key))
}

func NewTask(key client.ObjectKey) *Task {
	return &Task{
		Key:    key,
		Object: &tektonv1beta1.Task{},
	}
}
