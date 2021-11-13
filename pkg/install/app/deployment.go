package app

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	webhookTLSDirPath     = "/var/run/secrets/puppet/relay/webhook-tls"
	jwtSigningKeyDirPath  = "/var/run/secrets/puppet/relay/jwt"
	jwtSigningKeyPath     = "/var/run/secrets/puppet/relay/jwt/private-key.pem"
	metadataAPITLSDirPath = "/var/run/secrets/puppet/relay/tls"
)

func ConfigureOperatorDeployment(od *OperatorDeps, dep *appsv1obj.Deployment) {
	core := od.Core.Object

	template := &dep.Object.Spec.Template.Spec
	template.ServiceAccountName = dep.Key.Name
	template.Affinity = core.Spec.Operator.Affinity

	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-vault", dep.Key.Name),
				},
			},
		},
		{
			Name: "vault-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-vault", dep.Key.Name),
					},
				},
			},
		},
	}

	if core.Spec.Operator.AdmissionWebhookServer != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "webhook-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: core.Spec.Operator.AdmissionWebhookServer.TLSSecretName,
				},
			},
		})
	}

	if core.Spec.Operator.LogStoragePVCName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: v1alpha1.StepLogStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: *core.Spec.Operator.LogStoragePVCName,
				},
			},
		})
	}

	signingKeySecretName := od.SigningKeysSecret.Key.Name
	if core.Spec.Operator.JWTSigningKeySecretName != nil {
		signingKeySecretName = *core.Spec.Operator.JWTSigningKeySecretName
	}

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "jwt-signing-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: signingKeySecretName,
			},
		},
	})

	template.NodeSelector = core.Spec.Operator.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	sc := corev1.Container{}

	ConfigureOperatorContainer(od.Core, &sc)

	template.Containers[0] = sc

	vac := corev1.Container{}
	ConfigureVaultAgentContainer(od.Core, &vac)

	template.Containers[1] = vac
}

func ConfigureOperatorContainer(coreobj *obj.Core, c *corev1.Container) {
	core := coreobj.Object

	c.Name = "operator"
	c.Image = core.Spec.Operator.Image
	c.ImagePullPolicy = core.Spec.Operator.ImagePullPolicy

	env := []corev1.EnvVar{{Name: "VAULT_ADDR", Value: "http://localhost:8200"}}

	if core.Spec.SentryDSNSecretName != nil {
		env = append(env, corev1.EnvVar{
			Name: "RELAY_OPERATOR_SENTRY_DSN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *core.Spec.SentryDSNSecretName,
					},
					Key: "dsn",
				},
			},
		})
	}

	if core.Spec.Operator.Env != nil {
		env = append(env, core.Spec.Operator.Env...)
	}

	c.Env = env

	cmd := []string{
		"relay-operator",
		"-environment",
		core.Spec.Environment,
		"-num-workers",
		strconv.Itoa(int(core.Spec.Operator.Workers)),
		"-jwt-signing-key-file",
		jwtSigningKeyPath,
		"-vault-transit-path",
		core.Spec.Vault.TransitPath,
		"-vault-transit-key",
		core.Spec.Vault.TransitKey,
		"-dynamic-rbac-binding",
	}

	if core.Spec.Operator.Standalone {
		cmd = append(cmd, "-standalone")
	}

	if core.Spec.Operator.MetricsEnabled {
		cmd = append(cmd, "-metrics-enabled", "-metrics-server-bind-addr", "0.0.0.0:3050")
	}

	if core.Spec.Operator.TenantSandboxingRuntimeClassName != nil {
		cmd = append(cmd,
			"-tenant-sandboxing",
			"-tenant-sandbox-runtime-class-name",
			*core.Spec.Operator.TenantSandboxingRuntimeClassName,
		)
	}

	var storageAddr string
	if core.Spec.Operator.StorageAddr != nil {
		storageAddr = *core.Spec.Operator.StorageAddr
	} else {
		if core.Spec.Operator.LogStoragePVCName != nil {
			addr := url.URL{
				Scheme: "file",
				Path:   filepath.Join("/", v1alpha1.StepLogStorageVolumeName),
			}

			storageAddr = addr.String()
		}
	}

	if storageAddr != "" {
		cmd = append(cmd,
			"-storage-addr",
			storageAddr,
		)
	}

	if core.Spec.SentryDSNSecretName != nil {
		cmd = append(cmd, "-sentry-dsn", "$(RELAY_OPERATOR_SENTRY_DSN)")
	}

	cmd = append(cmd, "-metadata-api-url", *core.Spec.MetadataAPI.URL)

	if core.Spec.Operator.AdmissionWebhookServer != nil {
		cmd = append(cmd,
			"-webhook-server-key-dir",
			webhookTLSDirPath,
		)
	}

	if core.Spec.Operator.ToolInjection != nil {
		cmd = append(cmd,
			"-trigger-tool-injection-pool",
			core.Spec.Operator.ToolInjection.TriggerPoolName,
		)
	}

	c.Command = cmd

	c.VolumeMounts = []corev1.VolumeMount{}

	if core.Spec.Operator.AdmissionWebhookServer != nil {
		c.Ports = append(c.Ports, corev1.ContainerPort{
			Name:          "webhook",
			ContainerPort: int32(443),
			Protocol:      corev1.ProtocolTCP,
		})

		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      "webhook-tls",
			ReadOnly:  true,
			MountPath: webhookTLSDirPath,
		})
	}

	if core.Spec.Operator.LogStoragePVCName != nil {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      v1alpha1.StepLogStorageVolumeName,
			MountPath: filepath.Join("/", v1alpha1.StepLogStorageVolumeName),
		})
	}

	c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
		Name:      "jwt-signing-key",
		ReadOnly:  true,
		MountPath: jwtSigningKeyDirPath,
	})
}

func ConfigureMetadataAPIDeployment(md *MetadataAPIDeps, dep *appsv1obj.Deployment) {
	core := md.Core.Object

	dep.Object.Spec.Replicas = &core.Spec.MetadataAPI.Replicas

	template := &dep.Object.Spec.Template.Spec
	template.ServiceAccountName = dep.Key.Name
	template.Affinity = core.Spec.MetadataAPI.Affinity

	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-vault", dep.Key.Name),
				},
			},
		},
		{
			Name: "vault-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-vault", dep.Key.Name),
					},
				},
			},
		},
	}

	if core.Spec.MetadataAPI.TLSSecretName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "tls-crt",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *core.Spec.MetadataAPI.TLSSecretName,
				},
			},
		})
	}

	template.NodeSelector = core.Spec.MetadataAPI.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	sc := corev1.Container{}

	ConfigureMetadataAPIContainer(md.Core, &sc)

	template.Containers[0] = sc

	vac := corev1.Container{}
	ConfigureVaultAgentContainer(md.Core, &vac)

	template.Containers[1] = vac
}

func ConfigureMetadataAPIContainer(coreobj *obj.Core, c *corev1.Container) {
	core := coreobj.Object

	c.Name = "metadata-api"
	c.Image = core.Spec.MetadataAPI.Image
	c.ImagePullPolicy = core.Spec.MetadataAPI.ImagePullPolicy

	lsURL := ""
	if core.Spec.LogService != nil {
		if core.Spec.MetadataAPI.LogServiceURL != nil {
			lsURL = *core.Spec.MetadataAPI.LogServiceURL
		}
	}

	env := []corev1.EnvVar{
		{Name: "VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_METADATA_API_ENVIRONMENT", Value: core.Spec.Environment},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_PATH", Value: core.Spec.Vault.TransitPath},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_KEY", Value: core.Spec.Vault.TransitKey},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_PATH", Value: core.Spec.MetadataAPI.VaultAuthPath},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_ROLE", Value: core.Spec.MetadataAPI.VaultAuthRole},
		{Name: "RELAY_METADATA_API_LOG_SERVICE_URL", Value: lsURL},
		{Name: "RELAY_METADATA_API_STEP_METADATA_URL", Value: core.Spec.MetadataAPI.StepMetadataURL},
	}
	if core.Spec.Debug {
		env = append(env, corev1.EnvVar{Name: "RELAY_METADATA_API_DEBUG", Value: "true"})
	}

	if core.Spec.SentryDSNSecretName != nil {
		env = append(env, corev1.EnvVar{
			Name: "RELAY_METADATA_API_SENTRY_DSN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *core.Spec.SentryDSNSecretName,
					},
					Key: "dsn",
				},
			},
		})
	}

	if core.Spec.MetadataAPI.Env != nil {
		env = append(env, core.Spec.MetadataAPI.Env...)
	}

	c.Env = env

	c.Ports = []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: int32(7000),
			Protocol:      corev1.ProtocolTCP,
		},
	}

	probeScheme := corev1.URISchemeHTTP
	if core.Spec.MetadataAPI.TLSSecretName != nil {
		probeScheme = corev1.URISchemeHTTPS
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

	c.LivenessProbe = probe
	c.ReadinessProbe = probe

	if core.Spec.MetadataAPI.TLSSecretName != nil {
		c.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "tls-crt",
				MountPath: metadataAPITLSDirPath,
				ReadOnly:  true,
			},
		}
	}
}
