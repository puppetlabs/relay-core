package app

import (
	"net/url"
	"path/filepath"
	"strconv"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const jwtSigningKeyDirPath = "/var/run/secrets/puppet/relay/jwt"

func ConfigureOperatorDeployment(od *OperatorDeps, dep *appsv1obj.Deployment) {
	core := od.Core.Object

	if dep.Object.Labels == nil {
		dep.Object.Labels = make(map[string]string)
	}

	for k, v := range od.Labels {
		dep.Object.Labels[k] = v
	}

	dep.Object.Spec.Template.Labels = od.Labels
	dep.Object.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: od.Labels,
	}

	template := &dep.Object.Spec.Template.Spec
	template.ServiceAccountName = dep.Key.Name
	template.Affinity = core.Spec.Operator.Affinity

	template.Volumes = []corev1.Volume{
		{
			Name: "vault-agent-sa-token",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: od.VaultAgentDeps.TokenSecret.Key.Name,
				},
			},
		},
		{
			Name: "vault-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: od.VaultAgentDeps.ConfigMap.Key.Name,
					},
				},
			},
		},
	}

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "webhook-tls",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: helper.SuffixObjectKey(dep.Key, "certificate").Name,
			},
		},
	})

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

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "jwt-signing-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: core.Spec.JWTSigningKeyRef.Name,
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
		filepath.Join(jwtSigningKeyDirPath, core.Spec.JWTSigningKeyRef.PrivateKeyRef),
		"-vault-transit-path",
		core.Spec.Vault.TransitPath,
		"-vault-transit-key",
		core.Spec.Vault.TransitKey,
		"-dynamic-rbac-binding",
		"-webhook-server-key-dir",
		webhookTLSDirPath,
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

	if core.Spec.Operator.ToolInjection != nil {
		cmd = append(cmd,
			"-trigger-tool-injection-pool",
			core.Spec.Operator.ToolInjection.TriggerPoolName,
		)
	}

	c.Command = cmd

	c.VolumeMounts = []corev1.VolumeMount{}

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

func ConfigureOperatorWebhookService(od *OperatorDeps, svc *corev1obj.Service) {
	svc.Object.Spec.Selector = od.Labels

	svc.Object.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "webhook",
			Port:       int32(443),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("webhook"),
		},
	}
}
