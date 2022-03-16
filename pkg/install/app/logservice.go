package app

import (
	"fmt"
	"path/filepath"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DefaultLogServicePort = 7050
)

const (
	credentialsMountName = "google-application-credentials"
	credentialsMountPath = "/var/run/secrets/google"
)

func ConfigureLogServiceDeployment(ld *LogServiceDeps, dep *appsv1obj.Deployment) {
	core := ld.Core.Object

	dep.Object.Spec.Replicas = &core.Spec.LogService.Replicas

	if dep.Object.Labels == nil {
		dep.Object.Labels = make(map[string]string)
	}

	for k, v := range ld.Labels {
		dep.Object.Labels[k] = v
	}

	dep.Object.Spec.Template.Labels = ld.Labels
	dep.Object.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: ld.Labels,
	}

	template := &dep.Object.Spec.Template.Spec
	template.ServiceAccountName = dep.Key.Name
	template.Affinity = core.Spec.LogService.Affinity

	template.Volumes = ld.VaultAgentDeps.DeploymentVolumes()

	if core.Spec.LogService.CredentialsSecretKeyRef != nil {
		template.Volumes = append(template.Volumes,
			corev1.Volume{
				Name: credentialsMountName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: core.Spec.LogService.CredentialsSecretKeyRef.Name,
					},
				},
			},
		)
	}

	template.NodeSelector = core.Spec.LogService.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	sc := corev1.Container{}

	ConfigureLogServiceContainer(ld.Core, &sc)

	template.Containers[0] = sc
	template.Containers[1] = ld.VaultAgentDeps.SidecarContainer()
}

func ConfigureLogServiceContainer(coreobj *obj.Core, c *corev1.Container) {
	core := coreobj.Object

	c.Name = "log-service"
	c.Image = core.Spec.LogService.Image
	c.ImagePullPolicy = core.Spec.LogService.ImagePullPolicy

	env := []corev1.EnvVar{
		{Name: "RELAY_PLS_LISTEN_PORT", Value: fmt.Sprintf("%d", DefaultLogServicePort)},
		{Name: "RELAY_PLS_VAULT_ADDR", Value: "http://localhost:8200"},
		{Name: "RELAY_PLS_VAULT_ENGINE_MOUNT", Value: core.Spec.Vault.Engine.LogServicePath},
		{Name: "RELAY_PLS_PROJECT", Value: core.Spec.LogService.Project},
		{Name: "RELAY_PLS_DATASET", Value: core.Spec.LogService.Dataset},
		{Name: "RELAY_PLS_TABLE", Value: core.Spec.LogService.Table},
	}

	if core.Spec.LogService.CredentialsSecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name: "GOOGLE_APPLICATION_CREDENTIALS",
			Value: filepath.Join(
				credentialsMountPath,
				core.Spec.LogService.CredentialsSecretKeyRef.Key,
			),
		})
	}

	if core.Spec.Debug {
		env = append(env, corev1.EnvVar{Name: "RELAY_PLS_DEBUG", Value: "true"})
	}

	if core.Spec.LogService.Env != nil {
		env = append(env, core.Spec.LogService.Env...)
	}

	c.Env = env

	c.Ports = []corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: int32(DefaultLogServicePort),
			Protocol:      corev1.ProtocolTCP,
		},
	}

	if core.Spec.LogService.CredentialsSecretKeyRef != nil {
		c.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      credentialsMountName,
				MountPath: credentialsMountPath,
				ReadOnly:  true,
			},
		}
	}
}

func ConfigureLogServiceService(ld *LogServiceDeps, svc *corev1obj.Service) {
	svc.Object.Spec.Selector = ld.Labels

	svc.Object.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       int32(DefaultLogServicePort),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		},
	}
}
