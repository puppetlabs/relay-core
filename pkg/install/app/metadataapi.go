package app

import (
	"fmt"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ConfigureMetadataAPIDeployment(md *MetadataAPIDeps, dep *appsv1obj.Deployment) {
	core := md.Core.Object

	dep.Object.Spec.Replicas = &core.Spec.MetadataAPI.Replicas

	if dep.Object.Labels == nil {
		dep.Object.Labels = make(map[string]string)
	}

	for k, v := range md.Labels {
		dep.Object.Labels[k] = v
	}

	dep.Object.Spec.Template.Labels = md.Labels
	dep.Object.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: md.Labels,
	}

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
	// if core.Spec.LogService != nil {
	// 	if core.Spec.MetadataAPI.LogServiceURL != nil {
	// 		lsURL = *core.Spec.MetadataAPI.LogServiceURL
	// 	}
	// }
	if core.Spec.MetadataAPI.LogServiceURL != nil {
		lsURL = *core.Spec.MetadataAPI.LogServiceURL
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

func ConfigureMetadataAPIService(md *MetadataAPIDeps, svc *corev1obj.Service) {
	svc.Object.Spec.Selector = md.Labels

	port := int32(80)
	if md.Core.Object.Spec.MetadataAPI.TLSSecretName != nil {
		port = int32(443)
	}

	svc.Object.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       port,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		},
	}
}
