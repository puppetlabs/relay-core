package app

import (
	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureVaultConfigJob(vj *VaultConfigJobs, cm *corev1obj.ConfigMap, job *batchv1obj.Job) {
	template := &job.Object.Spec.Template

	vc := corev1.Container{}
	configureVaultConfigBaseContainer(vj, &vc)

	vc.Command = []string{
		"/bin/sh",
		"-c",
		vaultConfigureScript,
	}

	template.Spec.Containers = []corev1.Container{vc}
	template.Spec.RestartPolicy = "OnFailure"

	job.Object.Spec.Template = *template
}

func ConfigureVaultConfigUnsealJob(vj *VaultConfigJobs, job *batchv1obj.Job) {
	template := &job.Object.Spec.Template

	vc := corev1.Container{}
	configureVaultConfigBaseContainer(vj, &vc)

	cmd := "vault operator unseal ${VAULT_UNSEAL_KEY}"
	vc.Command = []string{"/bin/sh", "-c", cmd}

	template.Spec.Containers = []corev1.Container{vc}
	template.Spec.RestartPolicy = "Never"

	job.Object.Spec.Template = *template
}

func configureVaultConfigBaseContainer(vj *VaultConfigJobs, c *corev1.Container) {
	core := vj.Core.Object

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
	}

	tokenEnv, _ := vj.Auth.TokenEnvVar()
	c.Env = append(c.Env, tokenEnv)

	if unsealKeyEnv, ok := vj.Auth.UnsealKeyEnvVar(); ok {
		c.Env = append(c.Env, unsealKeyEnv)
	}
}

var (
	vaultConfigureScript = "vault status"
	// vaultConfigureScript = `
// apk add --no-cache jq gettext
// vault plugin register \
// -sha256=2455ad9450e415efb42fd28c1436f4bf7e377524be5e534e55e6658b8ef56bd2 \
// -command=vault-plugin-secrets-oauthapp-v3.0.0-beta.3-linux-amd64 \
// secret oauthapp
// vault auth enable -path=kubernetes kubernetes
// vault write auth/kubernetes/config \
// kubernetes_host="https://kubernetes.default.svc" \
// kubernetes_ca_cert="${VAULT_CA_CERT}" \
// token_reviewer_jwt="${VAULT_JWT_TOKEN}"
// vault auth enable -path=jwt-tenants jwt
// vault write ${JWT_AUTH_PATH}/config \
// jwt_supported_algs="RS256,RS512" \
// jwt_validation_pubkeys="${JWT_SIGNING_PUBLIC_KEY}"
// export AUTH_JWT_ACCESSOR="$( vault auth list -format=json | jq -r '."jwt-tenants/".accessor' )"
// vault secrets enable -path=${LOG_SERVICE_PATH} kv-v2
// vault secrets enable -path=${TENANT_PATH} kv-v2
// vault secrets enable -path=${TRANSIT_PATH} transit
// vault write ${TRANSIT_PATH}/keys/${TRANSIT_KEY} derived=true
// vault policy write ${OPERATOR_POLICY} /vault-policy-config/operator.hcl
// vault policy write ${METADATA_API_POLICY} /vault-policy-config/metadata-api.hcl
// vault policy write ${LOG_SERVICE_POLICY} /vault-policy-config/log-service.hcl
// envsubst '$AUTH_JWT_ACCESSOR' < /vault-policy-config/metadata-api-tenant.hcl > /tmp/metadata-api-tenant.hcl
// vault policy write ${METADATA_API_TENANT_POLICY} /tmp/metadata-api-tenant.hcl
// vault write auth/kubernetes/role/${OPERATOR_POLICY} \
// bound_service_account_names=${OPERATOR_SERVICE_ACCOUNT_NAME} \
// bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
// ttl=24h policies=${OPERATOR_POLICY}
// vault write auth/kubernetes/role/${METADATA_API_POLICY} \
// bound_service_account_names=${METADATA_API_SERVICE_ACCOUNT_NAME} \
// bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
// ttl=24h policies=${METADATA_API_POLICY}
// vault write auth/kubernetes/role/${LOG_SERVICE_POLICY} \
// bound_service_account_names=${LOG_SERVICE_SERVICE_ACCOUNT_NAME} \
// bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
// ttl=24h \
// policies=${LOG_SERVICE_POLICY}
// vault write ${JWT_AUTH_PATH}/role/${JWT_AUTH_ROLE} - < /vault-policy-config/metadata-api-tenant-config.json
// `
)
