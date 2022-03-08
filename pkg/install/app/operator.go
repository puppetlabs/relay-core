package app

import (
	"context"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const jwtSigningKeyDirPath = "/var/run/secrets/puppet/relay/jwt"

type operatorDeployment struct {
	*appsv1obj.Deployment

	core           *obj.Core
	vaultAgentDeps *VaultAgentDeps
}

func (d *operatorDeployment) Configure(_ context.Context) error {
	core := d.core.Object
	conf := core.Spec.Operator
	dep := d.Object

	dep.Spec.Template.Labels = dep.GetLabels()
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: dep.GetLabels(),
	}

	template := &dep.Spec.Template.Spec
	template.ServiceAccountName = conf.ServiceAccountName
	template.Affinity = conf.Affinity
	template.Volumes = d.vaultAgentDeps.DeploymentVolumes()

	template.Volumes = append(template.Volumes, corev1.Volume{
		Name: "webhook-tls",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: helper.SuffixObjectKey(d.Key, "certificate").Name,
			},
		},
	})

	if conf.LogStoragePVCName != nil {
		template.Volumes = append(template.Volumes, corev1.Volume{
			Name: v1alpha1.StepLogStorageVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: *conf.LogStoragePVCName,
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

	template.NodeSelector = conf.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 2)
	}

	template.Containers[0] = d.container()
	template.Containers[1] = d.vaultAgentDeps.SidecarContainer()

	return nil
}

func (d *operatorDeployment) container() corev1.Container {
	core := d.core.Object
	conf := core.Spec.Operator

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

	if conf.Env != nil {
		env = append(env, conf.Env...)
	}

	cmd := []string{
		"relay-operator",
		"-environment",
		core.Spec.Environment,
		"-num-workers",
		strconv.Itoa(int(conf.Workers)),
		"-jwt-signing-key-file",
		filepath.Join(jwtSigningKeyDirPath, core.Spec.JWTSigningKeyRef.PrivateKeyRef),
		"-vault-transit-path",
		core.Spec.Vault.Engine.TransitPath,
		"-vault-transit-key",
		core.Spec.Vault.Engine.TransitKey,
		"-dynamic-rbac-binding",
		"-webhook-server-key-dir",
		webhookTLSDirPath,
	}

	if conf.Standalone {
		cmd = append(cmd, "-standalone")
	}

	if conf.MetricsEnabled {
		cmd = append(cmd, "-metrics-enabled", "-metrics-server-bind-addr", "0.0.0.0:3050")
	}

	if conf.TenantSandboxingRuntimeClassName != nil {
		cmd = append(cmd,
			"-tenant-sandboxing",
			"-tenant-sandbox-runtime-class-name",
			*conf.TenantSandboxingRuntimeClassName,
		)
	}

	var storageAddr string
	if conf.StorageAddr != nil {
		storageAddr = *conf.StorageAddr
	} else {
		if conf.LogStoragePVCName != nil {
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

	c := corev1.Container{
		Name:            "operator",
		Image:           conf.Image,
		ImagePullPolicy: conf.ImagePullPolicy,
		Command:         cmd,
		Env:             env,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "webhook-tls",
				ReadOnly:  true,
				MountPath: webhookTLSDirPath,
			},
			{
				Name:      "jwt-signing-key",
				ReadOnly:  true,
				MountPath: jwtSigningKeyDirPath,
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "webhook",
				ContainerPort: int32(443),
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	if conf.LogStoragePVCName != nil {
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      v1alpha1.StepLogStorageVolumeName,
			MountPath: filepath.Join("/", v1alpha1.StepLogStorageVolumeName),
		})
	}

	return c
}

func newOperatorDeployment(key client.ObjectKey, core *obj.Core, vad *VaultAgentDeps) *operatorDeployment {
	return &operatorDeployment{
		Deployment:     appsv1obj.NewDeployment(key),
		core:           core,
		vaultAgentDeps: vad,
	}
}

type operatorWebhookService struct {
	*corev1obj.Service

	deployment *appsv1obj.Deployment
}

func (s *operatorWebhookService) Configure(_ context.Context) error {
	svc := s.Object
	svc.Spec.Selector = s.deployment.Object.GetLabels()

	svc.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "webhook",
			Port:       int32(443),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("webhook"),
		},
	}

	return nil
}

func newOperatorWebhookService(dep *operatorDeployment) *operatorWebhookService {
	return &operatorWebhookService{
		Service:    corev1obj.NewService(helper.SuffixObjectKey(dep.Key, "webhook")),
		deployment: dep.Deployment,
	}
}
