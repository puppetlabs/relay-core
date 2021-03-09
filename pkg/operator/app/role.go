package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func ConfigureMetadataAPIRole(role *rbacv1obj.Role, immutableConfigMap, mutableConfigMap *corev1obj.ConfigMap) {
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
