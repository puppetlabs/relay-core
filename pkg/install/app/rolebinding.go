package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func ConfigureRoleBinding(sa *corev1obj.ServiceAccount, rb *rbacv1obj.RoleBinding) {
	rb.Object.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     rbacv1obj.RoleKind.Kind,
		Name:     rb.Key.Name,
	}

	found := false
	for _, subject := range rb.Object.Subjects {
		if subject.Kind == corev1obj.ServiceAccountKind.Kind &&
			subject.Name == sa.Key.Name &&
			subject.Namespace == sa.Key.Namespace {
			found = true
		}
	}

	if !found {
		rb.Object.Subjects = append(rb.Object.Subjects,
			rbacv1.Subject{
				Kind:      corev1obj.ServiceAccountKind.Kind,
				Name:      sa.Key.Name,
				Namespace: sa.Key.Namespace,
			},
		)
	}
}
