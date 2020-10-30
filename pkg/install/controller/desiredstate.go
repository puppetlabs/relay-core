package controller

import (
	"fmt"
	"net/url"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type component string

func (c component) String() string {
	return string(c)
}

const (
	componentOperator    component = "operator"
	componentMetadataAPI component = "metadata-api"
)

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

func (m *vaultAgentManager) deploymentVolumes(deployment *appsv1.Deployment) {
	template := &deployment.Spec.Template.Spec

	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-vault", deployment.Name),
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

func (m *vaultAgentManager) serviceAccountTokenSecret(sa *corev1.ServiceAccount, secret *corev1.Secret) {
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	secret.Annotations["kubernetes.io/service-account.name"] = sa.Name
	secret.Type = corev1.SecretTypeServiceAccountToken
}

func (m *vaultAgentManager) getRole() string {
	role := fmt.Sprintf("%s-%s", m.rc.Name, m.component.String())

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

func baseLabels(relayCore *installerv1alpha1.RelayCore) map[string]string {
	return map[string]string{
		"install.relay.sh/relay-core":  relayCore.Name,
		"app.kubernetes.io/name":       "relay-operator",
		"app.kubernetes.io/managed-by": "relay-install-operator",
	}
}
