package obj

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
