package obj

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Role struct {
	Key    client.ObjectKey
	Object *rbacv1.Role
}

var _ Persister = &Role{}
var _ Loader = &Role{}
var _ Ownable = &Role{}
var _ LabelAnnotatableFrom = &Role{}

func (r *Role) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, r.Key, r.Object)
}

func (r *Role) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, r.Key, r.Object)
}

func (r *Role) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&r.Object.ObjectMeta, ref)
}

func (r *Role) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&r.Object.ObjectMeta, from)
}

func NewRole(key client.ObjectKey) *Role {
	return &Role{
		Key:    key,
		Object: &rbacv1.Role{},
	}
}

func ConfigureMetadataAPIRole(role *Role, immutableConfigMap, mutableConfigMap *ConfigMap) {
	role.Object.Rules = []rbacv1.PolicyRule{
		{
			APIGroups:     []string{""},
			Resources:     []string{"configmaps"},
			ResourceNames: []string{immutableConfigMap.Key.Name},
			Verbs:         []string{"get"},
		},
		{
			APIGroups:     []string{""},
			Resources:     []string{"configmaps"},
			ResourceNames: []string{mutableConfigMap.Key.Name},
			Verbs:         []string{"get", "update"},
		},
	}
}
