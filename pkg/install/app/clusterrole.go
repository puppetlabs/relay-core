package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func ConfigureOperatorClusterRole(cr *rbacv1obj.ClusterRole) {
	cr.Object.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "pods", "serviceaccounts", "secrets", "limitranges"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"tekton.dev"},
			Resources: []string{"pipelineruns", "taskruns", "pipelines", "tasks", "conditions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles", "rolebindings"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"networkpolicies"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"relay.sh"},
			Resources: []string{"runs", "runs/status", "tenants", "tenants/status", "webhooktriggers", "webhooktriggers/status", "workflows", "workflows/status"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{"serving.knative.dev"},
			Resources: []string{"revisions", "services"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
}

func ConfigureOperatorDelegateClusterRole(cr *rbacv1obj.ClusterRole) {
	cr.Object.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods/log"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "serviceaccounts", "secrets", "limitranges"},
			Verbs:     []string{"create", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles", "rolebindings"},
			Verbs:     []string{"create", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"networkpolicies"},
			Verbs:     []string{"create", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"tekton.dev"},
			Resources: []string{"pipelineruns", "taskruns", "pipelines", "tasks", "conditions"},
			Verbs:     []string{"create", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"serving.knative.dev"},
			Resources: []string{"revisions", "services"},
			Verbs:     []string{"create", "update", "patch", "delete"},
		},
	}
}

func ConfigureMetadataAPIClusterRole(cr *rbacv1obj.ClusterRole) {
	cr.Object.Rules = []rbacv1.PolicyRule{
		{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"get"}},
		{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
		{APIGroups: []string{"tekton.dev"}, Resources: []string{"conditions"}, Verbs: []string{"get", "list"}},
	}
}

func ConfigureClusterRoleBinding(sa *corev1obj.ServiceAccount, crb *rbacv1obj.ClusterRoleBinding) {
	ConfigureClusterRoleBindingWithRoleRef(sa, crb,
		rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     rbacv1obj.ClusterRoleKind.Kind,
			Name:     crb.Name,
		},
	)
}

func ConfigureClusterRoleBindingWithRoleRef(sa *corev1obj.ServiceAccount, crb *rbacv1obj.ClusterRoleBinding, rr rbacv1.RoleRef) {
	crb.Object.RoleRef = rr

	found := false
	for _, subject := range crb.Object.Subjects {
		if subject.Kind == corev1obj.ServiceAccountKind.Kind &&
			subject.Name == sa.Key.Name &&
			subject.Namespace == sa.Key.Namespace {
			found = true
		}
	}

	if !found {
		crb.Object.Subjects = append(crb.Object.Subjects,
			rbacv1.Subject{
				Kind:      corev1obj.ServiceAccountKind.Kind,
				Name:      sa.Key.Name,
				Namespace: sa.Key.Namespace,
			},
		)
	}
}

func ConfigureWebhookCertificateControllerClusterRole(cr *rbacv1obj.ClusterRole) {
	cr.Object.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"create", "update", "list", "get", "watch"},
		},
		{
			APIGroups: []string{"admissionregistration.k8s.io"},
			Resources: []string{"mutatingwebhookconfigurations"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
	}
}
