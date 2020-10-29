package controller

import (
	"fmt"
	"net/url"
	"strconv"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/install/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type component string

func (c component) String() string {
	return string(c)
}

const (
	componentOperator    component = "operator"
	componentMetadataAPI component = "metadata-api"
)

type operatorStateManager struct {
	rc                *installerv1alpha1.RelayCore
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

func (m *operatorStateManager) deployment(deployment *appsv1.Deployment, vaultTokenSecretName string) {
	setDeploymentLabels(m.baseLabels, deployment)

	template := &deployment.Spec.Template.Spec

	template.Affinity = m.rc.Spec.Operator.Affinity
	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: vaultTokenSecretName,
				},
			},
		},
		{
			Name: "vault-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-vault", deployment.Name),
					},
				},
			},
		},
	}

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

func newOperatorStateManager(rc *installerv1alpha1.RelayCore, labels map[string]string) *operatorStateManager {
	m := &operatorStateManager{
		rc:                rc,
		vaultAgentManager: newVaultAgentManager(rc, componentOperator),
		baseLabels: map[string]string{
			"app.kubernetes.io/component": componentOperator.String(),
		},
	}

	for k, v := range labels {
		m.baseLabels[k] = v
	}

	return m
}

type metadataAPIStateManager struct {
	rc                *installerv1alpha1.RelayCore
	vaultAgentManager *vaultAgentManager
	baseLabels        map[string]string
}

func (m *metadataAPIStateManager) deployment(deployment *appsv1.Deployment, vaultTokenSecretName string) {
	setDeploymentLabels(m.baseLabels, deployment)

	deployment.Spec.Replicas = &m.rc.Spec.MetadataAPI.Replicas

	template := &deployment.Spec.Template.Spec

	template.Affinity = m.rc.Spec.MetadataAPI.Affinity
	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: vaultTokenSecretName,
				},
			},
		},
		{
			Name: "vault-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-vault", deployment.Name),
					},
				},
			},
		},
	}

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
	env := []corev1.EnvVar{
		{Name: "VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_PATH", Value: m.rc.Spec.Vault.TransitPath},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_KEY", Value: m.rc.Spec.Vault.TransitKey},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_PATH", Value: m.rc.Spec.MetadataAPI.VaultAuthPath},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_ROLE", Value: m.rc.Spec.MetadataAPI.VaultAuthRole},
		{Name: "RELAY_METADATA_API_STEP_METADATA_URL", Value: m.rc.Spec.MetadataAPI.StepMetadataURL},
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

	container.LivenessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromString("http"),
				Scheme: probeScheme,
			},
		},
	}

	container.ReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromString("http"),
				Scheme: probeScheme,
			},
		},
	}

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

func newMetadataAPIStateManager(rc *installerv1alpha1.RelayCore, labels map[string]string) *metadataAPIStateManager {
	m := &metadataAPIStateManager{
		rc:                rc,
		vaultAgentManager: newVaultAgentManager(rc, componentOperator),
		baseLabels: map[string]string{
			"app.kubernetes.io/component": componentMetadataAPI.String(),
		},
	}

	for k, v := range labels {
		m.baseLabels[k] = v
	}

	return m
}

type vaultAgentManager struct {
	rc        *installerv1alpha1.RelayCore
	component component
}

func (m *vaultAgentManager) sidecarContainer(container *corev1.Container) {
	container.Name = "vault"
	container.Image = m.rc.Spec.Vault.Sidecar.Image
	container.ImagePullPolicy = m.rc.Spec.Vault.Sidecar.ImagePullPolicy
	container.Command = []string{
		"vault",
		"agent",
		"-config=/var/run/vault/config/agent.hcl",
	}
	container.Resources = m.rc.Spec.Vault.Sidecar.Resources

	container.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "vault-agent-config",
			ReadOnly:  true,
			MountPath: vaultAgentConfigDirPath,
		},
		{
			Name:      "vault-agent-sa-token",
			ReadOnly:  true,
			MountPath: vaultAgentSATokenPath,
		},
	}
}

func (m *vaultAgentManager) configMap(configMap *corev1.ConfigMap) {
	configFmt := `auto_auth {
      method "kubernetes" {
        mount_path = "auth/kubernetes"
        config     = {
          role       = "%s"
          token_path = "/var/run/secrets/kubernetes.io/serviceaccount@vault/token"
        }
      }
    }

    cache {
      use_auto_auth_token = true
    }

    listener "tcp" {
      address     = "127.0.0.1:8200"
      tls_disable = true
    }

    vault {
      address = "%s"
    }`

	config := fmt.Sprintf(configFmt, m.getRole(), m.rc.Spec.Vault.Sidecar.ServerAddr)

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	configMap.Data["agent.hcl"] = config
}

func (m *vaultAgentManager) getRole() string {
	role := fmt.Sprintf("%s-vault-%s", m.rc.Name, m.component.String())

	switch m.component {
	case componentOperator:
		if m.rc.Spec.Operator.VaultAgentRole != nil {
			role = *m.rc.Spec.Operator.VaultAgentRole
		}
	case componentMetadataAPI:
		if m.rc.Spec.MetadataAPI.VaultAgentRole != nil {
			role = *m.rc.Spec.MetadataAPI.VaultAgentRole
		}
	}

	return role
}

func newVaultAgentManager(rc *installerv1alpha1.RelayCore, component component) *vaultAgentManager {
	return &vaultAgentManager{
		rc:        rc,
		component: component,
	}
}

func setDeploymentLabels(labels map[string]string, deployment *appsv1.Deployment) {
	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}

	for k, v := range labels {
		deployment.Labels[k] = v
	}

	deployment.Spec.Template.Labels = labels
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
}

func metadataAPIURL(rc *installerv1alpha1.RelayCore) string {
	if rc.Spec.MetadataAPI.URL != nil {
		return *rc.Spec.MetadataAPI.URL
	}

	scheme := "http"
	if rc.Spec.MetadataAPI.TLSSecretName != nil {
		scheme = "https"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s-metadata-api.%s.svc.cluster.local", rc.Name, rc.Namespace),
	}

	return u.String()
}
