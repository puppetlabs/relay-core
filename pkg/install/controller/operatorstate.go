package controller

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/go-logr/logr"
	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/install/jwt"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type operatorObjects struct {
	deployment                     appsv1.Deployment
	webhookService                 corev1.Service
	serviceAccount                 corev1.ServiceAccount
	signingKeysSecret              corev1.Secret
	clusterRole                    rbacv1.ClusterRole
	clusterRoleBinding             rbacv1.ClusterRoleBinding
	delegateClusterRole            rbacv1.ClusterRole
	mutatingWebhookConfig          admissionv1.MutatingWebhookConfiguration
	vaultConfigMap                 corev1.ConfigMap
	vaultServiceAccount            corev1.ServiceAccount
	vaultServiceAccountTokenSecret corev1.Secret
}

func newOperatorObjects(rc *installerv1alpha1.RelayCore) *operatorObjects {
	name := fmt.Sprintf("%s-operator", rc.Name)
	vaultName := fmt.Sprintf("%s-vault", name)
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: rc.Namespace}
	vaultObjectMeta := metav1.ObjectMeta{Name: vaultName, Namespace: rc.Namespace}
	signingKeysSecretObjectMeta := metav1.ObjectMeta{Name: fmt.Sprintf("%s-signing-keys", name), Namespace: rc.Namespace}

	return &operatorObjects{
		deployment:                     appsv1.Deployment{ObjectMeta: objectMeta},
		webhookService:                 corev1.Service{ObjectMeta: objectMeta},
		serviceAccount:                 corev1.ServiceAccount{ObjectMeta: objectMeta},
		signingKeysSecret:              corev1.Secret{ObjectMeta: signingKeysSecretObjectMeta},
		clusterRole:                    rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		clusterRoleBinding:             rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		delegateClusterRole:            rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-delegate", objectMeta.Name)}},
		mutatingWebhookConfig:          admissionv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: name}},
		vaultConfigMap:                 corev1.ConfigMap{ObjectMeta: vaultObjectMeta},
		vaultServiceAccount:            corev1.ServiceAccount{ObjectMeta: vaultObjectMeta},
		vaultServiceAccountTokenSecret: corev1.Secret{ObjectMeta: vaultObjectMeta},
	}
}

type operatorStateManager struct {
	client.Client
	log               logr.Logger
	objects           *operatorObjects
	rc                *installerv1alpha1.RelayCore
	scheme            *runtime.Scheme
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

func (m *operatorStateManager) reconcile(ctx context.Context) error {
	log := m.log.WithValues("application", "relay-operator")

	log.Info("processing vault ConfigMap")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultConfigMap, func() error {
		m.vaultAgentManager.configMap(&m.objects.vaultConfigMap)

		return ctrl.SetControllerReference(m.rc, &m.objects.vaultConfigMap, m.scheme)
	}); err != nil {
		return err
	}

	log.Info("processing vault ServiceAccount")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultServiceAccount, func() error {
		return ctrl.SetControllerReference(m.rc, &m.objects.vaultServiceAccount, m.scheme)
	}); err != nil {
		return err
	}

	m.rc.Status.Vault.OperatorServiceAccount = m.objects.vaultServiceAccount.Name

	log.Info("processing vault ServiceAccount token Secret")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.vaultServiceAccountTokenSecret, func() error {
		m.vaultAgentManager.serviceAccountTokenSecret(
			&m.objects.vaultServiceAccount,
			&m.objects.vaultServiceAccountTokenSecret,
		)

		return ctrl.SetControllerReference(m.rc, &m.objects.vaultServiceAccountTokenSecret, m.scheme)
	}); err != nil {
		return err
	}

	log.Info("processing ServiceAccount")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.serviceAccount, func() error {
		return ctrl.SetControllerReference(m.rc, &m.objects.serviceAccount, m.scheme)
	}); err != nil {
		return err
	}

	m.rc.Status.OperatorServiceAccount = m.objects.serviceAccount.Name

	log.Info("processing ClusterRole")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.clusterRole, func() error {
		m.clusterRole(&m.objects.clusterRole)

		return nil
	}); err != nil {
		return err
	}

	log.Info("processing ClusterRoleBinding")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.clusterRoleBinding, func() error {
		m.clusterRoleBinding(&m.objects.clusterRoleBinding)

		return nil
	}); err != nil {
		return err
	}

	log.Info("processing DelegateClusterRole")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.delegateClusterRole, func() error {
		m.delegateClusterRole(&m.objects.delegateClusterRole)

		return nil
	}); err != nil {
		return err
	}

	if m.rc.Spec.Operator.GenerateJWTSigningKey {
		log.Info("generation of jwt signing keys is enabled")
		log.Info("processing jwt signing keys Secret")
		if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.signingKeysSecret, func() error {
			if m.objects.signingKeysSecret.CreationTimestamp.IsZero() {
				if err := m.signingKeysSecret(&m.objects.signingKeysSecret); err != nil {
					return err
				}
			}

			return ctrl.SetControllerReference(m.rc, &m.objects.signingKeysSecret, m.scheme)
		}); err != nil {
			return err
		}

		m.rc.Status.Vault.JWTSigningKeySecret = m.objects.signingKeysSecret.Name
	}

	log.Info("processing Deployment")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.deployment, func() error {
		m.deployment(&m.objects.deployment)

		return ctrl.SetControllerReference(m.rc, &m.objects.deployment, m.scheme)
	}); err != nil {
		return err
	}

	if m.rc.Spec.Operator.AdmissionWebhookServer != nil {
		log.Info("processing webhook Service")
		if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.webhookService, func() error {
			m.webhookService(&m.objects.webhookService)

			return ctrl.SetControllerReference(m.rc, &m.objects.webhookService, m.scheme)
		}); err != nil {
			return err
		}

		var caSecret *corev1.Secret
		if m.rc.Spec.Operator.AdmissionWebhookServer.CABundleSecretName != nil {
			log.Info("ca bundle secret name is set")
			log.Info("looking up ca bundle Secret", "name", *m.rc.Spec.Operator.AdmissionWebhookServer.CABundleSecretName)
			caSecretKey := client.ObjectKey{
				Name:      *m.rc.Spec.Operator.AdmissionWebhookServer.CABundleSecretName,
				Namespace: m.rc.Namespace,
			}

			caSecret = &corev1.Secret{}
			if err := m.Get(ctx, caSecretKey, caSecret); err != nil {
				return err
			}

			if _, ok := caSecret.Data["ca.crt"]; !ok {
				return fmt.Errorf("ca bundle does not exist in secret: %s", caSecret.Name)
			}
		}

		log.Info("processing MutatingWebhookConfiguration")
		if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.mutatingWebhookConfig, func() error {
			m.mutatingWebhookConfig(caSecret, &m.objects.mutatingWebhookConfig)

			return nil
		}); err != nil {
			return err
		}
	}

	m.rc.Status.Vault.OperatorRole = m.vaultAgentManager.getRole()

	return nil
}

func (m *operatorStateManager) deployment(deployment *appsv1.Deployment) {
	setDeploymentLabels(m.baseLabels, deployment)

	template := &deployment.Spec.Template.Spec

	template.ServiceAccountName = deployment.Name
	template.Affinity = m.rc.Spec.Operator.Affinity

	m.vaultAgentManager.deploymentVolumes(deployment)

	if m.rc.Spec.Operator.AdmissionWebhookServer != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "webhook-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: m.rc.Spec.Operator.AdmissionWebhookServer.TLSSecretName,
				},
			},
		})
	}

	if m.rc.Spec.Operator.LogStoragePVCName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: installerv1alpha1.StepLogStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: *m.rc.Spec.Operator.LogStoragePVCName,
				},
			},
		})
	}

	signingKeySecretName := m.objects.signingKeysSecret.Name
	if m.rc.Spec.Operator.JWTSigningKeySecretName != nil {
		signingKeySecretName = *m.rc.Spec.Operator.JWTSigningKeySecretName
	}

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "jwt-signing-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: signingKeySecretName,
			},
		},
	})

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
		"-tool-injection-image",
		m.rc.Spec.Operator.ToolInjection.Image,
		"-num-workers",
		strconv.Itoa(int(m.rc.Spec.Operator.Workers)),
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

	storageAddr := m.storageAddress()
	if storageAddr != "" {
		cmd = append(cmd,
			"-storage-addr",
			storageAddr,
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

	if m.rc.Spec.Operator.AdmissionWebhookServer != nil {
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

func (m *operatorStateManager) webhookService(svc *corev1.Service) {
	svc.Spec.Selector = m.baseLabels

	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "webhook",
			Port:       int32(443),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("webhook"),
		},
	}
}

func (m *operatorStateManager) serverContainer(container *corev1.Container) {
	container.Name = componentOperator.String()
	container.Image = m.rc.Spec.Operator.Image
	container.ImagePullPolicy = m.rc.Spec.Operator.ImagePullPolicy
	container.Env = m.deploymentEnv()
	container.Command = m.deploymentCommand()

	container.VolumeMounts = []corev1.VolumeMount{}

	if m.rc.Spec.Operator.AdmissionWebhookServer != nil {
		container.Ports = append(container.Ports, corev1.ContainerPort{
			Name:          "webhook",
			ContainerPort: int32(443),
			Protocol:      corev1.ProtocolTCP,
		})

		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "webhook-tls",
			ReadOnly:  true,
			MountPath: webhookTLSDirPath,
		})
	}

	if m.rc.Spec.Operator.LogStoragePVCName != nil {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      installerv1alpha1.StepLogStorageVolumeName,
			MountPath: filepath.Join("/", installerv1alpha1.StepLogStorageVolumeName),
		})
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      "jwt-signing-key",
		ReadOnly:  true,
		MountPath: jwtSigningKeyDirPath,
	})
}

func (m *operatorStateManager) signingKeysSecret(sec *corev1.Secret) error {
	pair, err := jwt.GenerateSigningKeys()
	if err != nil {
		return err
	}

	if sec.Data == nil {
		sec.Data = make(map[string][]byte)
	}

	sec.Data["private-key.pem"] = pair.PrivateKey
	sec.Data["public-key.pem"] = pair.PublicKey

	return nil
}

func (m *operatorStateManager) clusterRole(clusterRole *rbacv1.ClusterRole) {
	clusterRole.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces", "persistentvolumes", "persistentvolumeclaims"},
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
			APIGroups: []string{"batch", "extensions"},
			Resources: []string{"jobs"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
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

func (m *operatorStateManager) delegateClusterRole(clusterRole *rbacv1.ClusterRole) {
	clusterRole.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods/log"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "serviceaccounts", "secrets", "limitranges", "persistentvolumeclaims"},
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
			Resources: []string{"services"},
			Verbs:     []string{"create", "update", "patch", "delete"},
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

func (m *operatorStateManager) mutatingWebhookConfig(caSecret *corev1.Secret, mwh *admissionv1.MutatingWebhookConfiguration) {
	var (
		podEnforcementPath = "/mutate/pod-enforcement"
		volumeClaimPath    = "/mutate/volume-claim"
	)

	var caCert []byte
	if caSecret != nil {
		caCert = caSecret.Data["ca.crt"]
	}

	failurePolicy := admissionv1.Fail
	sideEffects := admissionv1.SideEffectClassNone
	reinvocationPolicy := admissionv1.IfNeededReinvocationPolicy

	matchLabels := map[string]string{"controller.relay.sh/tenant-workload": "true"}

	mwh.Webhooks = []admissionv1.MutatingWebhook{
		{
			AdmissionReviewVersions: []string{"v1beta1"},
			Name:                    fmt.Sprintf("%s-pod-enforcement.admission.controller.relay.sh", mwh.Name),
			ClientConfig: admissionv1.WebhookClientConfig{
				CABundle: caCert,
				Service: &admissionv1.ServiceReference{
					Name:      m.objects.webhookService.Name,
					Namespace: m.objects.webhookService.Namespace,
					Path:      &podEnforcementPath,
				},
			},
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update},
					Rule: admissionv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			FailurePolicy:      &failurePolicy,
			SideEffects:        &sideEffects,
			ReinvocationPolicy: &reinvocationPolicy,
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
		{
			AdmissionReviewVersions: []string{"v1beta1"},
			Name:                    fmt.Sprintf("%s-volume-claim.admission.controller.relay.sh", mwh.Name),
			ClientConfig: admissionv1.WebhookClientConfig{
				CABundle: caCert,
				Service: &admissionv1.ServiceReference{
					Name:      m.objects.webhookService.Name,
					Namespace: m.objects.webhookService.Namespace,
					Path:      &volumeClaimPath,
				},
			},
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update},
					Rule: admissionv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			FailurePolicy:      &failurePolicy,
			SideEffects:        &sideEffects,
			ReinvocationPolicy: &reinvocationPolicy,
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

func (m *operatorStateManager) storageAddress() string {
	if m.rc.Spec.Operator.StorageAddr != nil {
		return *m.rc.Spec.Operator.StorageAddr
	}

	if m.rc.Spec.Operator.LogStoragePVCName != nil {
		addr := url.URL{
			Scheme: "file",
			Path:   filepath.Join("/", installerv1alpha1.StepLogStorageVolumeName),
		}

		return addr.String()
	}

	return ""
}

func newOperatorStateManager(rc *installerv1alpha1.RelayCore, r *RelayCoreReconciler, log logr.Logger) *operatorStateManager {
	m := &operatorStateManager{
		Client:            r.Client,
		log:               log,
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
