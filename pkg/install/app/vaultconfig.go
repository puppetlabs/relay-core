package app

import (
	"path/filepath"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureVaultConfigJob(vd *VaultConfigDeps, job *batchv1obj.Job) {
	const vaultConfigDirName = "vault-config"

	template := &job.Object.Spec.Template.Spec

	vc := corev1.Container{}

	vc.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      vaultConfigDirName,
			MountPath: filepath.Join("/", vaultConfigDirName),
		},
	}

	configureVaultConfigBaseContainer(vd, &vc)

	vc.Command = []string{
		"/bin/sh",
		filepath.Join("/", vaultConfigDirName, "run-config"),
	}

	template.Containers = []corev1.Container{vc}
	template.RestartPolicy = "OnFailure"

	template.Volumes = []corev1.Volume{
		{
			Name: vaultConfigDirName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: vd.Core.Object.Spec.Vault.ConfigMapRef.LocalObjectReference,
				},
			},
		},
	}

	job.Object.Spec.Template.Spec = *template
}

func ConfigureVaultConfigUnsealJob(vd *VaultConfigDeps, job *batchv1obj.Job) {
	template := &job.Object.Spec.Template.Spec

	vc := corev1.Container{}
	configureVaultConfigBaseContainer(vd, &vc)

	cmd := "vault operator unseal ${VAULT_UNSEAL_KEY}"
	vc.Command = []string{"/bin/sh", "-c", cmd}

	template.Containers = []corev1.Container{vc}
	template.RestartPolicy = "Never"

	job.Object.Spec.Template.Spec = *template
}

func configureVaultConfigBaseContainer(vd *VaultConfigDeps, c *corev1.Container) {
	core := vd.Core.Object

	c.Name = "vault-client"
	c.Image = core.Spec.Vault.Sidecar.Image
	c.ImagePullPolicy = core.Spec.Vault.Sidecar.ImagePullPolicy

	vault := core.Spec.Vault
	operator := core.Spec.Operator
	metadataAPI := core.Spec.MetadataAPI
	logService := core.Spec.LogService

	c.Env = []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "VAULT_ADDR",
			Value: core.Spec.Vault.Sidecar.ServerAddr,
		},
		corev1.EnvVar{
			Name:  "LOG_SERVICE_PATH",
			Value: vault.LogServicePath,
		},
		corev1.EnvVar{
			Name:  "TENANT_PATH",
			Value: vault.TenantPath,
		},
		corev1.EnvVar{
			Name:  "TRANSIT_PATH",
			Value: vault.TransitPath,
		},
		corev1.EnvVar{
			Name:  "TRANSIT_KEY",
			Value: vault.TransitKey,
		},
		corev1.EnvVar{
			Name:  "OPERATOR_ROLE",
			Value: operator.VaultAgentRole,
		},
		corev1.EnvVar{
			Name:  "OPERATOR_SERVICE_ACCOUNT_NAME",
			Value: operator.ServiceAccountName,
		},
		corev1.EnvVar{
			Name:  "OPERATOR_SERVICE_ACCOUNT_NAMESPACE",
			Value: core.Namespace,
		},
		corev1.EnvVar{
			Name:  "METADATA_API_ROLE",
			Value: metadataAPI.VaultAgentRole,
		},
		corev1.EnvVar{
			Name:  "METADATA_API_SERVICE_ACCOUNT_NAME",
			Value: metadataAPI.ServiceAccountName,
		},
		corev1.EnvVar{
			Name:  "METADATA_API_SERVICE_ACCOUNT_NAMESPACE",
			Value: core.Namespace,
		},
		corev1.EnvVar{
			Name:  "METADATA_API_TENANT_ROLE",
			Value: norm.AnyDNSLabelNameSuffixed(metadataAPI.VaultAgentRole, "-tenant"),
		},
		corev1.EnvVar{
			Name:  "LOG_SERVICE_ROLE",
			Value: logService.VaultAgentRole,
		},
		corev1.EnvVar{
			Name:  "LOG_SERVICE_SERVICE_ACCOUNT_NAME",
			Value: logService.ServiceAccountName,
		},
		corev1.EnvVar{
			Name:  "LOG_SERVICE_SERVICE_ACCOUNT_NAMESPACE",
			Value: core.Namespace,
		},
		corev1.EnvVar{
			Name:  "JWT_AUTH_ROLE",
			Value: metadataAPI.VaultAuthRole,
		},
		corev1.EnvVar{
			Name:  "JWT_AUTH_PATH",
			Value: metadataAPI.VaultAuthPath,
		},
		corev1.EnvVar{
			Name: "JWT_PUBLIC_SIGNING_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  core.Spec.JWTSigningKeyRef.PublicKeyRef,
					LocalObjectReference: core.Spec.JWTSigningKeyRef.LocalObjectReference,
				},
			},
		},
	}

	tokenEnv, _ := vd.Auth.TokenEnvVar()
	c.Env = append(c.Env, tokenEnv)

	if unsealKeyEnv, ok := vd.Auth.UnsealKeyEnvVar(); ok {
		c.Env = append(c.Env, unsealKeyEnv)
	}
}
