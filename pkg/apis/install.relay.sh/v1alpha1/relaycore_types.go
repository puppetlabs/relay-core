/*
Copyright 2020 Puppet, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StepLogStorageVolumeName = "step-log-storage"
)

// RelayCoreSpec defines the desired state of RelayCore
type RelayCoreSpec struct {
	// Environment is the environment this instance is running in.
	//
	// +kubebuilder:default="dev"
	// +optional
	Environment string `json:"environment,omitempty"`

	// Debug enabled debug logging and tools where possible.
	//
	// +kubebuilder:default=false
	// +optional
	Debug bool `json:"debug"`

	// JWTSigningKeys is the secret and keys that hold a JWT signing key pair
	// for the workflow run key signing operations with vault. This secret must
	// have 2 fields for a public and private key pair. If this field is not
	// set, then signings key will be generated automatically.
	//
	// +optional
	JWTSigningKeyRef *JWTSigningKeySource `json:"jwtSigningKeys,omitempty"`

	// LogService is the configuration for the log service.
	//
	// +optional
	LogService *LogServiceConfig `json:"logService,omitempty"`

	// Operator is the configuration for the workflow run operator.
	Operator OperatorConfig `json:"operator"`

	// MetadataAPI is the configuration for the step metadata-api server.
	MetadataAPI MetadataAPIConfig `json:"metadataAPI"`

	// Vault is the configuration for accessing vault.
	Vault VaultConfig `json:"vault"`

	// SentryDSNSecretName is the secret that holds the DSN address for Sentry
	// error and stacktrace collection. The secret object MUST have a data
	// field called "dsn".
	//
	// +optional
	SentryDSNSecretName *string `json:"sentryDSNSecretName,omitempty"`
}

type JWTSigningKeySource struct {
	corev1.LocalObjectReference `json:",inline"`

	PrivateKeyRef string `json:"privateKeyRef,omitempty"`
	PublicKeyRef  string `json:"publicKeyRef,omitempty"`
}

// LogServiceConfig is the configuration for the relay-log-service deployment
type LogServiceConfig struct {
	// Image is the container image to use for the log service.
	//
	// +kubebuilder:default="relaysh/relay-pls:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy instructs the cluster when it should attempt to pull the
	// container image.
	//
	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Env is the slice of environment variables to use when launching the log
	// service.
	//
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// NodeSelector instructs the cluster how to choose a node to run the log
	// service pods.
	//
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the service account to use to run this service's pods.
	// This is the service account that is also handed to Vault for Kubernetes Auth.
	//
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Affinity is an optional set of affinity constraints to apply to operator
	// pods.
	//
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Replicas is the number of pods to run for this server.
	//
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +kubebuilder:default="log-service"
	// +optional
	VaultAgentRole string `json:"vaultAgentRole,omitempty"`

	// CredentialsSecretKeyRef is the secret and key to use for the log service
	// cloud credentials
	CredentialsSecretKeyRef *corev1.SecretKeySelector `json:"credentialsSecretKeyRef,omitempty"`

	// Project is the BigQuery project to use for logging.
	Project string `json:"project,omitempty"`

	// Dataset is the BigQuery dataset to use for logging.
	Dataset string `json:"dataset,omitempty"`

	// Project is the BigQuery table to use for logging.
	Table string `json:"table,omitempty"`
}

// OperatorConfig is the configuration for the relay-operator deployment
type OperatorConfig struct {
	// +kubebuilder:default="us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-operator:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// MetricsEnabled enables the metrics server for the operator deployment
	// and creates a service that can be used to scrape those metrics.
	//
	// +kubebuilder:default=false
	// +optional
	MetricsEnabled bool `json:"metricsEnabled"`

	// TenantSandboxingRuntimeClassName sets the class to use for sandboxing
	// application kernels on tenant pods. If this is set to a value, then
	// tenant sandboxing is enabled in the operator.
	// TODO: should this be an kubebuilder enum of supported runtimes?
	//
	// +optional
	TenantSandboxingRuntimeClassName *string `json:"tenantSandboxingRuntimeClassName,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the service account to use to run this service's pods.
	// This is the service account that is also handed to Vault for Kubernetes Auth.
	//
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Affinity is an optional set of affinity constraints to apply to operator
	// pods.
	//
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +kubebuilder:default=false
	// +optional
	Standalone bool `json:"standalone"`

	// StorageAddr is the storage address URI for log storage.
	//
	// +optional
	StorageAddr *string `json:"storageAddr,omitempty"`

	// TenantNamespace is the Kubernetes namespace the operator should look for
	// tenant workloads on.
	//
	// +optional
	TenantNamespace *string `json:"tenantNamespace,omitempty"`

	// LogStoragePVCName is the name of a PVC to store logs in. This field is
	// here to support the development environment and may be removed at a
	// later date when the PLS implementation is rolled in.
	//
	// DEPRECATED
	//
	// +optional
	LogStoragePVCName *string `json:"logStoragePVCName,omitempty"`

	// Workers is the number of workers the operator should run to process
	// workflows
	//
	// +kubebuilder:default=2
	// +optional
	Workers int32 `json:"workers,omitempty"`

	// ToolInjection is the configuration for the entrypointer and tool
	// injection runtime tooling.
	//
	// +optional
	ToolInjection *ToolInjectionConfig `json:"toolInjection,omitempty"`

	// AdmissionWebhookServer is the configuration for the
	// admissionregistration webhook server.
	//
	//
	// +kubebuilder:default={certificateControllerImagePullPolicy: "IfNotPresent"}
	// +optional
	AdmissionWebhookServer *AdmissionWebhookServerConfig `json:"admissionWebhookServer,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +kubebuilder:default="operator"
	// +optional
	VaultAgentRole string `json:"vaultAgentRole,omitempty"`
}

type AdmissionWebhookServerConfig struct {
	// CertificateControllerImage is the image to use for the certificate
	// controller that managers the TLS certificates for the operator's webhook
	// server
	//
	// +kubebuilder:default="us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-operator-webhook-certificate-controller:latest"
	// +optional
	CertificateControllerImage string `json:"certificateControllerImage,omitempty"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	CertificateControllerImagePullPolicy corev1.PullPolicy `json:"certificateControllerImagePullPolicy,omitempty"`

	// Domain is the domain to use as a suffix for the webhook subdomain.
	// Example: admission.controller.example.com
	Domain string `json:"domain,omitempty"`

	// NamespaceSelector is the map of labels to use in the NamespaceSelector
	// section of the MutatingWebhooks.
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

// MetadataAPIConfig is the configuration for the relay-metadata-api deployment
type MetadataAPIConfig struct {
	// +kubebuilder:default="us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-metadata-api:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the service account to use to run this service's pods.
	// This is the service account that is also handed to Vault for Kubernetes Auth.
	//
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Affinity is an optional set of affinity constraints to apply to
	// metadata-api pods.
	//
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// TLSSecretName is the name of the secret that holds the TLS certificate
	// for enabling HTTPS on the metadata-api server. The secret object MUST
	// have two data fields called "tls.key" and "tls.crt".
	//
	// +optional
	TLSSecretName *string `json:"tlsSecretName,omitempty"`

	// URL is the URL of the metadata-api that will be used by workflows and
	// the operator. This defaults to:
	// http(s)://<RelayCore.Name>-metadata-api.<RelayCore.Namespace>.svc.cluster.local
	//
	// +optional
	URL *string `json:"url,omitempty"`

	// LogServiceURL is the URL of the service used to persist log messages.
	//
	// +optional
	LogServiceURL *string `json:"logServiceURL,omitempty"`

	// StepMetadataURL is the URL to use to fetch step metadata for schema
	// validation.
	//
	// +kubebuilder:default="https://relay.sh/step-metadata.json"
	// +optional
	StepMetadataURL string `json:"stepMetadataURL,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +kubebuilder:default="metadata-api"
	// +optional
	VaultAgentRole string `json:"vaultAgentRole,omitempty"`

	// +kubebuilder:default="tenant"
	// +optional
	VaultAuthRole string `json:"vaultAuthRole,omitempty"`

	// +kubebuilder:default="auth/jwt-tenants"
	// +optional
	VaultAuthPath string `json:"vaultAuthPath,omitempty"`
}

type VaultConfig struct {
	// Auth provides credentials for vault server authentication.
	//
	// +optional
	Auth *VaultAuthConfig `json:"auth"`

	// Engine provides the configuration for the internal vault engine.
	Engine VaultEngineConfig `json:"engine"`

	// Server provides the configuration for the vault server.
	Server VaultServerConfig `json:"server"`

	// Sidecar is the configuration for the vault sidecar containers.
	Sidecar VaultSidecarConfig `json:"sidecar"`
}

type VaultEngineConfig struct {
	// +kubebuilder:default="us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-operator-vault-init:latest"
	// +optional
	VaultInitializationImage string `json:"vaultInitializationImage,omitempty"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	VaultInitializationImagePullPolicy corev1.PullPolicy `json:"vaultInitializationImagePullPolicy,omitempty"`

	// +kubebuilder:default="pls"
	// +optional
	LogServicePath string `json:"logServicePath,omitempty"`

	// +kubebuilder:default="metadata-api"
	// +optional
	TransitKey string `json:"transitKey,omitempty"`

	// +kubebuilder:default="transit-tenants"
	// +optional
	TransitPath string `json:"transitPath,omitempty"`

	// +kubebuilder:default="customers"
	// +optional
	TenantPath string `json:"tenantPath,omitempty"`

	// AuthDelegatorServiceAccount is the name of the service account that
	// should be used to give vault token review access for the kubernetes auth
	// method.
	//
	// +optional
	AuthDelegatorServiceAccountName string `json:"authDelegatorServiceAccountName,omitempty"`
}

type VaultServerConfig struct {
	// Address is the address to the vault server.
	//
	// +kubebuilder:default="http://vault:8200"
	// +optional
	Address string `json:"address,omitempty"`

	// BuiltIn optionally instantiates an internal vault deployment for use.
	//
	// +optional
	BuiltIn *VaultServerBuiltInConfig `json:"builtIn"`
}

type VaultServerBuiltInConfig struct {
	// +kubebuilder:default="vault:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Resources sets the resource requirements for the vault sidecar containers.
	//
	// +kubebuilder:default={limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ConfigMapRef is the reference to the config map that contains the
	// scripts and policies to configure vault with.
	//
	// +optional
	ConfigMapRef corev1.LocalObjectReference `json:"configMapRef,omitempty"`
}

type VaultSidecarConfig struct {
	// +kubebuilder:default="vault:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Resources sets the resource requirements for the vault sidecar containers.
	//
	// +kubebuilder:default={limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type VaultAuthConfig struct {
	// Token is the token to use for vault server authentication when
	// configuring engine mounts and policies for relay-core components.
	//
	// +optional
	Token *VaultAuthData `json:"token,omitempty"`

	// UnsealKey enables a Job to unseal a vault server.
	// This Job only supports a singular unseal key, so
	// servers that require multiple keys will not be unsealed.
	//
	// +optional
	UnsealKey *VaultAuthData `json:"unsealKey,omitempty"`
}

type VaultAuthData struct {
	// Value provides vault server authentication data.
	//
	// +optional
	Value string `json:"token,omitempty"`

	// ValueFrom allows vault server auth data to be provided by another source
	// such as a Secret.
	//
	// +optional
	ValueFrom *VaultAuthSource `json:"tokenFrom,omitempty"`
}

type VaultAuthSource struct {
	// SecretKeyRef selects an API token by looking up the value in a secret.
	//
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

type ToolInjectionConfig struct {
	// Image is the image to use for the relay tool injection.
	//
	// +kubebuilder:default="us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-runtime-tools:latest"
	// +optional
	Image string `json:"image"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type RelayCore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RelayCoreSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// RelayCoreList contains a list of RelayCore
type RelayCoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RelayCore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RelayCore{}, &RelayCoreList{})
}
