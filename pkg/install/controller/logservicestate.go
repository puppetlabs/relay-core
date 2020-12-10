package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type logServiceObjects struct {
	deployment                     appsv1.Deployment
	service                        corev1.Service
	serviceAccount                 corev1.ServiceAccount
	vaultConfigMap                 corev1.ConfigMap
	vaultServiceAccount            corev1.ServiceAccount
	vaultServiceAccountTokenSecret corev1.Secret
}

func newLogServiceObjects(rc *installerv1alpha1.RelayCore) *logServiceObjects {
	name := fmt.Sprintf("%s-log-service", rc.Name)
	vaultName := fmt.Sprintf("%s-vault", name)
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: rc.Namespace}
	vaultObjectMeta := metav1.ObjectMeta{Name: vaultName, Namespace: rc.Namespace}

	return &logServiceObjects{
		deployment:                     appsv1.Deployment{ObjectMeta: objectMeta},
		service:                        corev1.Service{ObjectMeta: objectMeta},
		serviceAccount:                 corev1.ServiceAccount{ObjectMeta: objectMeta},
		vaultConfigMap:                 corev1.ConfigMap{ObjectMeta: vaultObjectMeta},
		vaultServiceAccount:            corev1.ServiceAccount{ObjectMeta: vaultObjectMeta},
		vaultServiceAccountTokenSecret: corev1.Secret{ObjectMeta: vaultObjectMeta},
	}
}

type logServiceStateManager struct {
	client.Client
	objects           *logServiceObjects
	log               logr.Logger
	rc                *installerv1alpha1.RelayCore
	scheme            *runtime.Scheme
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

func (m *logServiceStateManager) reconcile(ctx context.Context) error {
	log := m.log.WithValues("application", "relay-log-service")

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

	m.rc.Status.Vault.LogServiceServiceAccount = m.objects.vaultServiceAccount.Name

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

	m.rc.Status.LogServiceServiceAccount = m.objects.serviceAccount.Name

	log.Info("processing Deployment")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.deployment, func() error {
		m.deployment(&m.objects.deployment)

		return ctrl.SetControllerReference(m.rc, &m.objects.deployment, m.scheme)
	}); err != nil {
		return err
	}

	m.rc.Status.Vault.LogServiceRole = m.vaultAgentManager.getRole()

	log.Info("processing Service")
	if _, err := ctrl.CreateOrUpdate(ctx, m, &m.objects.service, func() error {
		m.httpService(&m.objects.service)

		return ctrl.SetControllerReference(m.rc, &m.objects.service, m.scheme)
	}); err != nil {
		return err
	}

	return nil
}

func (m *logServiceStateManager) deployment(deployment *appsv1.Deployment) {
	setDeploymentLabels(m.baseLabels, deployment)

	deployment.Spec.Replicas = &m.rc.Spec.LogService.Replicas

	template := &deployment.Spec.Template.Spec

	template.ServiceAccountName = deployment.Name
	template.Affinity = m.rc.Spec.LogService.Affinity

	m.vaultAgentManager.deploymentVolumes(deployment)
	m.deploymentVolumes(deployment)

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	template.NodeSelector = m.rc.Spec.LogService.NodeSelector

	logServiceContainer := corev1.Container{}
	m.serverContainer(&logServiceContainer)

	template.Containers[0] = logServiceContainer

	vaultSidecar := corev1.Container{}
	m.vaultAgentManager.sidecarContainer(&vaultSidecar)

	template.Containers[1] = vaultSidecar
}

func (m *logServiceStateManager) deploymentVolumes(deployment *appsv1.Deployment) {
	template := &deployment.Spec.Template.Spec

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "google-application-credentials",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: m.rc.Spec.LogService.CredentialsSecretName,
			},
		},
	})
}

func (m *logServiceStateManager) deploymentEnv() []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/var/run/secrets/google/key.json"},
		{Name: "RELAY_PLS_LISTEN_PORT", Value: "7050"},
		{Name: "RELAY_PLS_VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_PLS_PROJECT", Value: m.rc.Spec.LogService.Project},
		{Name: "RELAY_PLS_DATASET", Value: m.rc.Spec.LogService.Dataset},
		{Name: "RELAY_PLS_TABLE", Value: m.rc.Spec.LogService.Table},
	}

	if m.rc.Spec.Debug {
		env = append(env, corev1.EnvVar{Name: "RELAY_PLS_DEBUG", Value: "true"})
	}

	if m.rc.Spec.LogService.Env != nil {
		env = append(env, m.rc.Spec.LogService.Env...)
	}

	return env
}

func (m *logServiceStateManager) serverContainer(container *corev1.Container) {
	container.Name = componentLogService.String()
	container.Image = m.rc.Spec.LogService.Image
	container.ImagePullPolicy = m.rc.Spec.LogService.ImagePullPolicy
	container.Env = m.deploymentEnv()

	container.Ports = []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: int32(7050),
			Protocol:      corev1.ProtocolTCP,
		},
	}

	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "google-application-credentials",
			MountPath: "/var/run/secrets/google",
			ReadOnly:  true,
		},
	}
}

func (m *logServiceStateManager) httpService(svc *corev1.Service) {
	svc.Spec.Selector = m.baseLabels

	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       int32(7050),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		},
	}
}

func newLogServiceStateManager(rc *installerv1alpha1.RelayCore, r *RelayCoreReconciler, log logr.Logger) *logServiceStateManager {
	m := &logServiceStateManager{
		Client:            r.Client,
		objects:           newLogServiceObjects(rc),
		log:               log,
		rc:                rc,
		scheme:            r.Scheme,
		vaultAgentManager: newVaultAgentManager(rc, componentLogService),
		baseLabels: map[string]string{
			"app.kubernetes.io/component": componentLogService.String(),
		},
	}

	for k, v := range baseLabels(rc) {
		m.baseLabels[k] = v
	}

	return m
}
