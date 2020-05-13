package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Namespace struct {
	Name   string
	Object *corev1.Namespace
}

var _ Loader = &Namespace{}

func (n *Namespace) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, client.ObjectKey{Name: n.Name}, n.Object)
}

func NewNamespace(name string) *Namespace {
	return &Namespace{
		Name:   name,
		Object: &corev1.Namespace{},
	}
}
