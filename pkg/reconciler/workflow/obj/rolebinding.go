package obj

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RoleBinding struct {
	Key    client.ObjectKey
	Object *rbacv1.RoleBinding
}

var _ Persister = &RoleBinding{}
var _ Loader = &RoleBinding{}
var _ Ownable = &RoleBinding{}

func (rb *RoleBinding) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, rb.Key, rb.Object)
}

func (rb *RoleBinding) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, rb.Key, rb.Object)
}

func (rb *RoleBinding) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&rb.Object.ObjectMeta, ref)
}

func NewRoleBinding(key client.ObjectKey) *RoleBinding {
	return &RoleBinding{
		Key:    key,
		Object: &rbacv1.RoleBinding{},
	}
}

func ConfigureMetadataAPIRoleBinding(rb *RoleBinding, sa *ServiceAccount, role *Role) {
	rb.Object.RoleRef = rbacv1.RoleRef{
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
		Name:     role.Key.Name,
	}
	rb.Object.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Namespace: sa.Key.Namespace,
			Name:      sa.Key.Name,
		},
	}
}
