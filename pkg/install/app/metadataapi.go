package app

import (
	"context"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type metadataAPIDeployment struct {
	*appsv1obj.Deployment

	core           *obj.Core
	vaultAgentDeps *VaultAgentDeps
}

func (d *metadataAPIDeployment) Configure(_ context.Context) error {
	core := d.core.Object
	conf := core.Spec.MetadataAPI
	dep := d.Object

	dep.Spec.Replicas = &conf.Replicas

	dep.Spec.Template.Labels = dep.GetLabels()
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: dep.GetLabels(),
	}

	template := &dep.Spec.Template.Spec
	template.ServiceAccountName = conf.ServiceAccountName
	template.Affinity = conf.Affinity
	template.Volumes = d.vaultAgentDeps.DeploymentVolumes()

	if conf.TLSSecretName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: "tls-crt",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *conf.TLSSecretName,
				},
			},
		})
	}

	template.NodeSelector = conf.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	sc := corev1.Container{}
	d.configureContainer(&sc)
	template.Containers[0] = sc
	template.Containers[1] = d.vaultAgentDeps.SidecarContainer()

	return nil
}

func (d *metadataAPIDeployment) configureContainer(c *corev1.Container) {
	core := d.core.Object
	conf := core.Spec.MetadataAPI

	c.Name = "metadata-api"
	c.Image = conf.Image
	c.ImagePullPolicy = conf.ImagePullPolicy

	lsURL := ""
	if conf.LogServiceURL != nil {
		lsURL = *conf.LogServiceURL
	}

	env := []corev1.EnvVar{
		{Name: "VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_METADATA_API_ENVIRONMENT", Value: core.Spec.Environment},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_PATH", Value: core.Spec.Vault.TransitPath},
		{Name: "RELAY_METADATA_API_VAULT_TRANSIT_KEY", Value: core.Spec.Vault.TransitKey},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_PATH", Value: conf.VaultAuthPath},
		{Name: "RELAY_METADATA_API_VAULT_AUTH_ROLE", Value: conf.VaultAuthRole},
		{Name: "RELAY_METADATA_API_LOG_SERVICE_URL", Value: lsURL},
		{Name: "RELAY_METADATA_API_STEP_METADATA_URL", Value: conf.StepMetadataURL},
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

	if conf.Env != nil {
		env = append(env, conf.Env...)
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
	if conf.TLSSecretName != nil {
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

	if conf.TLSSecretName != nil {
		c.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "tls-crt",
				MountPath: metadataAPITLSDirPath,
				ReadOnly:  true,
			},
		}
	}
}

func newMetadataAPIDeployment(key client.ObjectKey, core *obj.Core, vad *VaultAgentDeps) *metadataAPIDeployment {
	return &metadataAPIDeployment{
		Deployment:     appsv1obj.NewDeployment(key),
		core:           core,
		vaultAgentDeps: vad,
	}
}

type metadataAPIService struct {
	*corev1obj.Service

	core       *obj.Core
	deployment *appsv1obj.Deployment
}

func (s *metadataAPIService) Configure(_ context.Context) error {
	core := s.core.Object
	conf := core.Spec.MetadataAPI
	svc := s.Object

	svc.Spec.Selector = s.deployment.Object.GetLabels()

	port := int32(80)
	if conf.TLSSecretName != nil {
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

	return nil
}

func newMetadataAPIService(core *obj.Core, dep *metadataAPIDeployment) *metadataAPIService {
	return &metadataAPIService{
		Service:    corev1obj.NewService(dep.Key),
		core:       core,
		deployment: dep.Deployment,
	}
}
