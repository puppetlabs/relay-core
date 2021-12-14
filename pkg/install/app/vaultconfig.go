package app

import (
	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureVaultConfigConfigMap(c *obj.Core, cm *corev1obj.ConfigMap) {

}

func ConfigureVaultConfigJob(vd *VaultConfigDeps, job *batchv1obj.Job) {
	template := &job.Object.Spec.Template

	vc := corev1.Container{}
	configureVaultConfigBaseContainer(vd, &vc)

	vc.Command = []string{
		"/bin/sh",
		"-c",
		vaultConfigureScript,
	}

	template.Spec.Containers = []corev1.Container{vc}
}

func ConfigureVaultConfigUnsealJob(vd *VaultConfigDeps, job *batchv1obj.Job) {

}

func configureVaultConfigBaseContainer(vd *VaultConfigDeps, c *corev1.Container) {
	core := vd.Core.Object

	c.Name = "vault-client"
	c.Image = core.Spec.Vault.Sidecar.Image
	c.ImagePullPolicy = core.Spec.Vault.Sidecar.ImagePullPolicy

	c.Env = []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "VAULT_ADDR",
			Value: core.Spec.Vault.Sidecar.ServerAddr,
		},
	}

	tokenEnv, _ := vd.Auth.TokenEnvVar()
	c.Env = append(c.Env, tokenEnv)

	if unsealKeyEnv, ok := vd.Auth.UnsealKeyEnvVar(); ok {
		c.Env = append(c.Env, unsealKeyEnv)
	}
}

var (
	vaultConfigureScript = `
apk add --no-cache jq gettext
vault plugin register \
	-sha256=2455ad9450e415efb42fd28c1436f4bf7e377524be5e534e55e6658b8ef56bd2 \
	-command=vault-plugin-secrets-oauthapp-v3.0.0-beta.3-linux-amd64 \
	secret oauthapp
vault auth enable -path=kubernetes kubernetes
vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc" \
    kubernetes_ca_cert="${VAULT_CA_CERT}" \
    token_reviewer_jwt="${VAULT_JWT_TOKEN}"
vault auth enable -path=jwt-tenants jwt
vault write ${JWT_AUTH_PATH}/config \
   jwt_supported_algs="RS256,RS512" \
   jwt_validation_pubkeys="${JWT_SIGNING_PUBLIC_KEY}"
export AUTH_JWT_ACCESSOR="$( vault auth list -format=json | jq -r '."jwt-tenants/".accessor' )"
vault secrets enable -path=${LOG_SERVICE_PATH} kv-v2
vault secrets enable -path=${TENANT_PATH} kv-v2
vault secrets enable -path=${TRANSIT_PATH} transit
vault write ${TRANSIT_PATH}/keys/${TRANSIT_KEY} derived=true
vault policy write ${OPERATOR_POLICY} /vault-policy-config/operator.hcl
vault policy write ${METADATA_API_POLICY} /vault-policy-config/metadata-api.hcl
vault policy write ${LOG_SERVICE_POLICY} /vault-policy-config/log-service.hcl
envsubst '$AUTH_JWT_ACCESSOR' < /vault-policy-config/metadata-api-tenant.hcl > /tmp/metadata-api-tenant.hcl
vault policy write ${METADATA_API_TENANT_POLICY} /tmp/metadata-api-tenant.hcl
vault write auth/kubernetes/role/${OPERATOR_POLICY} \
    bound_service_account_names=${OPERATOR_SERVICE_ACCOUNT_NAME} \
    bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
    ttl=24h policies=${OPERATOR_POLICY}
vault write auth/kubernetes/role/${METADATA_API_POLICY} \
    bound_service_account_names=${METADATA_API_SERVICE_ACCOUNT_NAME} \
    bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
	ttl=24h policies=${METADATA_API_POLICY}
vault write auth/kubernetes/role/${LOG_SERVICE_POLICY} \
    bound_service_account_names=${LOG_SERVICE_SERVICE_ACCOUNT_NAME} \
    bound_service_account_namespaces=${SERVICE_ACCOUNT_NAMESPACE} \
    ttl=24h \
    policies=${LOG_SERVICE_POLICY}
vault write ${JWT_AUTH_PATH}/role/${JWT_AUTH_ROLE} - < /vault-policy-config/metadata-api-tenant-config.json
`
)
