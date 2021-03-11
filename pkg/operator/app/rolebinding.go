package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func ConfigureMetadataAPIRoleBinding(rb *rbacv1obj.RoleBinding, sa *corev1obj.ServiceAccount, role *rbacv1obj.Role) {
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
