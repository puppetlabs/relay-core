package obj

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
