package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Namespace struct {
	Name   string
	Object *corev1.Namespace
}

var _ Persister = &Namespace{}
var _ Loader = &Namespace{}
var _ Deleter = &Namespace{}
var _ LabelAnnotatableFrom = &Namespace{}

func (n *Namespace) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, client.ObjectKey{Name: n.Name}, n.Object)
}

func (n *Namespace) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, client.ObjectKey{Name: n.Name}, n.Object)
}

func (n *Namespace) Delete(ctx context.Context, cl client.Client) (bool, error) {
	return DeleteIgnoreNotFound(ctx, cl, n.Object)
}

func (n *Namespace) Label(ctx context.Context, name, value string) {
	Label(&n.Object.ObjectMeta, name, value)
}

func (n *Namespace) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&n.Object.ObjectMeta, from)
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:   name,
		Object: &corev1.Namespace{},
	}
}
