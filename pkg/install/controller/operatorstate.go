package controller

import (
	"context"
	"fmt"
	"strconv"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type operatorObjects struct {
	deployment                     appsv1.Deployment
	serviceAccount                 corev1.ServiceAccount
	clusterRole                    rbacv1.ClusterRole
	clusterRoleBinding             rbacv1.ClusterRoleBinding
	vaultConfigMap                 corev1.ConfigMap
	vaultServiceAccount            corev1.ServiceAccount
	vaultServiceAccountTokenSecret corev1.Secret
}

func newOperatorObjects(rc *installerv1alpha1.RelayCore) *operatorObjects {
	name := fmt.Sprintf("%s-operator", rc.Name)
	vaultName := fmt.Sprintf("%s-vault", name)
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: rc.Namespace}
	vaultObjectMeta := metav1.ObjectMeta{Name: vaultName, Namespace: rc.Namespace}

	return &operatorObjects{
		deployment:                     appsv1.Deployment{ObjectMeta: objectMeta},
		serviceAccount:                 corev1.ServiceAccount{ObjectMeta: objectMeta},
		clusterRole:                    rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		clusterRoleBinding:             rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		vaultConfigMap:                 corev1.ConfigMap{ObjectMeta: vaultObjectMeta},
		vaultServiceAccount:            corev1.ServiceAccount{ObjectMeta: vaultObjectMeta},
		vaultServiceAccountTokenSecret: corev1.Secret{ObjectMeta: vaultObjectMeta},
	}
}

type operatorStateManager struct {
	client.Client
	objects           *operatorObjects
	rc                *installerv1alpha1.RelayCore
	scheme            *runtime.Scheme
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

// _, err = ctrl.CreateOrUpdate(ctx, r, &operatorDelegateClusterRole, func() error {
// 	osm.delegateClusterRole(&operatorDelegateClusterRole)

// 	// return ctrl.SetControllerReference(relayCore, &operatorDelegateClusterRole, r.Scheme)
// 	return nil
// })
// if err != nil {
// 	return ctrl.Result{}, err
// }

func (m *operatorStateManager) reconcile(ctx context.Context) error {
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultConfigMap, func() error {
		m.vaultAgentManager.configMap(&m.objects.vaultConfigMap)

		return ctrl.SetControllerReference(m.rc, &m.objects.vaultConfigMap, m.scheme)
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultServiceAccount, func() error {
		return ctrl.SetControllerReference(m.rc, &m.objects.vaultServiceAccount, m.scheme)
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultServiceAccountTokenSecret, func() error {
		m.vaultAgentManager.serviceAccountTokenSecret(
			&m.objects.vaultServiceAccount,
			&m.objects.vaultServiceAccountTokenSecret,
		)

		return ctrl.SetControllerReference(m.rc, &m.objects.vaultServiceAccountTokenSecret, m.scheme)
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.serviceAccount, func() error {
		return ctrl.SetControllerReference(m.rc, &m.objects.serviceAccount, m.scheme)
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.clusterRole, func() error {
		m.clusterRole(&m.objects.clusterRole)

		return nil
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.clusterRoleBinding, func() error {
		m.clusterRoleBinding(&m.objects.clusterRoleBinding)

		return nil
	}); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.deployment, func() error {
		m.deployment(&m.objects.deployment)

		return ctrl.SetControllerReference(m.rc, &m.objects.deployment, m.scheme)
	}); err != nil {
		return err
	}

	return nil
}

func (m *operatorStateManager) deployment(deployment *appsv1.Deployment) {
	setDeploymentLabels(m.baseLabels, deployment)

	template := &deployment.Spec.Template.Spec

	template.ServiceAccountName = deployment.Name
	template.Affinity = m.rc.Spec.Operator.Affinity

	m.vaultAgentManager.deploymentVolumes(deployment)

	if m.rc.Spec.Operator.WebhookTLSSecretName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "webhook-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *m.rc.Spec.Operator.WebhookTLSSecretName,
				},
			},
		})
	}

	if m.rc.Spec.Operator.JWTSigningKeySecretName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "jwt-signing-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *m.rc.Spec.Operator.JWTSigningKeySecretName,
				},
			},
		})
	}

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	template.NodeSelector = m.rc.Spec.Operator.NodeSelector

	operatorContainer := corev1.Container{}
	m.serverContainer(&operatorContainer)

	template.Containers[0] = operatorContainer

	vaultSidecar := corev1.Container{}
	m.vaultAgentManager.sidecarContainer(&vaultSidecar)

	template.Containers[1] = vaultSidecar
}

func (m *operatorStateManager) deploymentCommand() []string {
	cmd := []string{
		"relay-operator",
		"-environment",
		m.rc.Spec.Environment,
		"-storage-addr",
		m.rc.Spec.Operator.StorageAddr,
		"-tool-injection-image",
		m.rc.Spec.Operator.ToolInjection.Image,
		"-num-workers",
		strconv.Itoa(int(m.rc.Spec.Operator.Workers)),
		// TODO convert to generateJWTSigningKey field
		"-jwt-signing-key-file",
		jwtSigningKeyPath,
		"-vault-transit-path",
		m.rc.Spec.Vault.TransitPath,
		"-vault-transit-key",
		m.rc.Spec.Vault.TransitKey,
		"-dynamic-rbac-binding",
	}

	if m.rc.Spec.Operator.Standalone {
		cmd = append(cmd, "-standalone")
	}

	if m.rc.Spec.Operator.MetricsEnabled {
		cmd = append(cmd, "-metrics-enabled", "-metrics-server-bind-addr", "0.0.0.0:3050")
	}

	if m.rc.Spec.Operator.TenantSandboxingRuntimeClassName != nil {
		cmd = append(cmd,
			"-tenant-sandboxing",
			"-tenant-sandbox-runtime-class-name",
			*m.rc.Spec.Operator.TenantSandboxingRuntimeClassName,
		)
	}

	if m.rc.Spec.SentryDSNSecretName != nil {
		cmd = append(cmd,
			"-sentry-dsn",
			"$(RELAY_OPERATOR_SENTRY_DSN)",
		)
	}

	cmd = append(cmd,
		"-metadata-api-url",
		metadataAPIURL(m.rc),
	)

	if m.rc.Spec.Operator.WebhookTLSSecretName != nil {
		cmd = append(cmd,
			"-webhook-server-key-dir",
			webhookTLSDirPath,
		)
	}

	return cmd
}

func (m *operatorStateManager) deploymentEnv() []corev1.EnvVar {
	env := []corev1.EnvVar{{Name: "VAULT_ADDR", Value: "http://localhost:8200"}}

	if m.rc.Spec.SentryDSNSecretName != nil {
		env = append(env, corev1.EnvVar{
			Name: "RELAY_OPERATOR_SENTRY_DSN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *m.rc.Spec.SentryDSNSecretName,
					},
					Key: "dsn",
				},
			},
		})
	}

	if m.rc.Spec.Operator.Env != nil {
		env = append(env, m.rc.Spec.Operator.Env...)
	}

	return env
}

func (m *operatorStateManager) serverContainer(container *corev1.Container) {
	container.Name = componentOperator.String()
	container.Image = m.rc.Spec.Operator.Image
	container.ImagePullPolicy = m.rc.Spec.Operator.ImagePullPolicy
	container.Env = m.deploymentEnv()
	container.Command = m.deploymentCommand()

	container.VolumeMounts = []corev1.VolumeMount{}

	if m.rc.Spec.Operator.WebhookTLSSecretName != nil {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "webhook-tls",
			ReadOnly:  true,
			MountPath: webhookTLSDirPath,
		})
	}

	if m.rc.Spec.Operator.JWTSigningKeySecretName != nil {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "jwt-signing-key",
			ReadOnly:  true,
			MountPath: jwtSigningKeyDirPath,
		})
	}
}

func (m *operatorStateManager) metricsService(svc *corev1.Service) {
}

func (m *operatorStateManager) clusterRole(clusterRole *rbacv1.ClusterRole) {
	clusterRole.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces", "persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "pods", "serviceaccounts", "secrets", "limitranges", "persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"tekton.dev"},
			Resources: []string{"pipelineruns", "taskruns", "pipelines", "tasks", "conditions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"batch", "extensions"},
			Resources: []string{"jobs"},
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
			APIGroups: []string{"nebula.puppet.com"},
			Resources: []string{"workflowruns", "workflowruns/status"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{"relay.sh"},
			Resources: []string{"tenants", "tenants/status", "webhooktriggers", "webhooktriggers/status"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{"serving.knative.dev"},
			Resources: []string{"services"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
}

func (m *operatorStateManager) clusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) {
	clusterRoleBinding.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     clusterRoleBinding.Name,
	}

	clusterRoleBinding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      clusterRoleBinding.Name,
			Namespace: m.rc.Namespace,
		},
	}
}

// func (m *operatorStateManager) delegateClusterRole(clusterRole *rbacv1.ClusterRole) {
// 	clusterRole.Rules = []rbacv1.PolicyRule{
// 		{
// 			APIGroups: []string{""},
// 			Resources: []string{"pods/log"},
// 			Verbs:     []string{"get", "list", "watch"},
// 		},
// 		{
// 			APIGroups: []string{""},
// 			Resources: []string{"configmaps", "serviceaccounts", "secrets", "limitranges", "persistentvolumes", "persistentvolumeclaims"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 		{
// 			APIGroups: []string{"batch", "extensions"},
// 			Resources: []string{"jobs"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 		{
// 			APIGroups: []string{"rbac.authorization.k8s.io"},
// 			Resources: []string{"roles", "rolebindings"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 		{
// 			APIGroups: []string{"networking.k8s.io"},
// 			Resources: []string{"networkpolicies"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 		{
// 			APIGroups: []string{"tekton.dev"},
// 			Resources: []string{"pipelineruns", "taskruns", "pipelines", "tasks", "conditions"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 		{
// 			APIGroups: []string{"serving.knative.dev"},
// 			Resources: []string{"services"},
// 			Verbs:     []string{"create", "update", "patch", "delete"},
// 		},
// 	}
// }

func newOperatorStateManager(rc *installerv1alpha1.RelayCore, r *RelayCoreReconciler) *operatorStateManager {
	m := &operatorStateManager{
		Client:            r.Client,
		objects:           newOperatorObjects(rc),
		rc:                rc,
		scheme:            r.Scheme,
		vaultAgentManager: newVaultAgentManager(rc, componentOperator),
		baseLabels: map[string]string{
			"app.kubernetes.io/component": componentOperator.String(),
		},
	}

	for k, v := range baseLabels(rc) {
		m.baseLabels[k] = v
	}

	return m
}
