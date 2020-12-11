package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type metadataAPIObjects struct {
	deployment                     appsv1.Deployment
	service                        corev1.Service
	serviceAccount                 corev1.ServiceAccount
	clusterRole                    rbacv1.ClusterRole
	clusterRoleBinding             rbacv1.ClusterRoleBinding
	vaultConfigMap                 corev1.ConfigMap
	vaultServiceAccount            corev1.ServiceAccount
	vaultServiceAccountTokenSecret corev1.Secret
}

func newMetadataAPIObjects(rc *installerv1alpha1.RelayCore) *metadataAPIObjects {
	name := fmt.Sprintf("%s-metadata-api", rc.Name)
	vaultName := fmt.Sprintf("%s-vault", name)
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: rc.Namespace}
	vaultObjectMeta := metav1.ObjectMeta{Name: vaultName, Namespace: rc.Namespace}

	return &metadataAPIObjects{
		deployment:                     appsv1.Deployment{ObjectMeta: objectMeta},
		service:                        corev1.Service{ObjectMeta: objectMeta},
		serviceAccount:                 corev1.ServiceAccount{ObjectMeta: objectMeta},
		clusterRole:                    rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		clusterRoleBinding:             rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: objectMeta.Name}},
		vaultConfigMap:                 corev1.ConfigMap{ObjectMeta: vaultObjectMeta},
		vaultServiceAccount:            corev1.ServiceAccount{ObjectMeta: vaultObjectMeta},
		vaultServiceAccountTokenSecret: corev1.Secret{ObjectMeta: vaultObjectMeta},
	}
}

type metadataAPIStateManager struct {
	client.Client
	objects           *metadataAPIObjects
	log               logr.Logger
	rc                *installerv1alpha1.RelayCore
	scheme            *runtime.Scheme
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

func (m *metadataAPIStateManager) reconcile(ctx context.Context) error {
	log := m.log.WithValues("application", "relay-metadata-api")

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

	m.rc.Status.Vault.MetadataAPIServiceAccount = m.objects.vaultServiceAccount.Name

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

	m.rc.Status.MetadataAPIServiceAccount = m.objects.serviceAccount.Name

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

	log.Info("processing Deployment")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.deployment, func() error {
		m.deployment(&m.objects.deployment)

		return ctrl.SetControllerReference(m.rc, &m.objects.deployment, m.scheme)
	}); err != nil {
		return err
	}

	m.rc.Status.Vault.MetadataAPIRole = m.vaultAgentManager.getRole()

	log.Info("processing Service")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.service, func() error {
		m.httpService(&m.objects.service)

		return ctrl.SetControllerReference(m.rc, &m.objects.service, m.scheme)
	}); err != nil {
		return err
	}

	return nil
}

func (m *metadataAPIStateManager) deployment(deployment *appsv1.Deployment) {
	setDeploymentLabels(m.baseLabels, deployment)

	deployment.Spec.Replicas = &m.rc.Spec.MetadataAPI.Replicas

	template := &deployment.Spec.Template.Spec

	template.ServiceAccountName = deployment.Name
	template.Affinity = m.rc.Spec.MetadataAPI.Affinity

	m.vaultAgentManager.deploymentVolumes(deployment)

	if m.rc.Spec.MetadataAPI.TLSSecretName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "tls-crt",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *m.rc.Spec.MetadataAPI.TLSSecretName,
				},
			},
		})
	}

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	template.NodeSelector = m.rc.Spec.MetadataAPI.NodeSelector

	metadataAPIContainer := corev1.Container{}
	m.serverContainer(&metadataAPIContainer)

	template.Containers[0] = metadataAPIContainer

	vaultSidecar := corev1.Container{}
	m.vaultAgentManager.sidecarContainer(&vaultSidecar)

	template.Containers[1] = vaultSidecar
}

func (m *metadataAPIStateManager) deploymentEnv() []corev1.EnvVar {
	// FIXME Implement a better mechanism for service-to-service configuration
	lsURL := ""
	if m.rc.Spec.MetadataAPI.LogServiceEnabled {
		lsURL = m.rc.Spec.MetadataAPI.LogServiceURL
		if lsURL == "" {
			lsURL = logServiceURL(m.rc)
		}
	}

	env := []corev1.EnvVar{
		{Name: "VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_METADATA_API_ENVIRONMENT", Value: m.rc.Spec.Environment},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_PATH", Value: m.rc.Spec.Vault.TransitPath},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_KEY", Value: m.rc.Spec.Vault.TransitKey},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_PATH", Value: m.rc.Spec.MetadataAPI.VaultAuthPath},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_ROLE", Value: m.rc.Spec.MetadataAPI.VaultAuthRole},
		{Name: "RELAY_METADATA_API_LOG_SERVICE_URL", Value: lsURL},
		{Name: "RELAY_METADATA_API_STEP_METADATA_URL", Value: m.rc.Spec.MetadataAPI.StepMetadataURL},
	}

	if m.rc.Spec.Debug {
		env = append(env, corev1.EnvVar{Name: "RELAY_METADATA_API_DEBUG", Value: "true"})
	}

	if m.rc.Spec.SentryDSNSecretName != nil {
		env = append(env, corev1.EnvVar{
			Name: "RELAY_METADATA_API_SENTRY_DSN",
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

	if m.rc.Spec.MetadataAPI.Env != nil {
		env = append(env, m.rc.Spec.MetadataAPI.Env...)
	}

	return env
}

func (m *metadataAPIStateManager) serverContainer(container *corev1.Container) {
	probeScheme := corev1.URISchemeHTTP
	if m.rc.Spec.MetadataAPI.TLSSecretName != nil {
		probeScheme = corev1.URISchemeHTTPS
	}

	container.Name = componentMetadataAPI.String()
	container.Image = m.rc.Spec.MetadataAPI.Image
	container.ImagePullPolicy = m.rc.Spec.MetadataAPI.ImagePullPolicy
	container.Env = m.deploymentEnv()

	container.Ports = []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: int32(7000),
			Protocol:      corev1.ProtocolTCP,
		},
	}

	probe := &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromString("http"),
				Scheme: probeScheme,
			},
		},
	}

	container.LivenessProbe = probe
	container.ReadinessProbe = probe

	if m.rc.Spec.MetadataAPI.TLSSecretName != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "tls-crt",
				MountPath: metadataAPITLSDirPath,
				ReadOnly:  true,
			},
		}
	}
}

func (m *metadataAPIStateManager) httpService(svc *corev1.Service) {
	svc.Spec.Selector = m.baseLabels

	port := int32(80)
	if m.rc.Spec.MetadataAPI.TLSSecretName != nil {
		port = int32(443)
	}

	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       port,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		},
	}
}

func (m *metadataAPIStateManager) clusterRole(clusterRole *rbacv1.ClusterRole) {
	clusterRole.Rules = []rbacv1.PolicyRule{
		{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"get"}},
		{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
		{APIGroups: []string{"tekton.dev"}, Resources: []string{"conditions"}, Verbs: []string{"get", "list"}},
	}
}

func (m *metadataAPIStateManager) clusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) {
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

func newMetadataAPIStateManager(rc *installerv1alpha1.RelayCore, r *RelayCoreReconciler, log logr.Logger) *metadataAPIStateManager {
	m := &metadataAPIStateManager{
		Client:            r.Client,
		objects:           newMetadataAPIObjects(rc),
		log:               log,
		rc:                rc,
		scheme:            r.Scheme,
		vaultAgentManager: newVaultAgentManager(rc, componentMetadataAPI),
		baseLabels: map[string]string{
			"app.kubernetes.io/component": componentMetadataAPI.String(),
		},
	}

	for k, v := range baseLabels(rc) {
		m.baseLabels[k] = v
	}

	return m
}
