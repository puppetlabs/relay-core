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

type Status string

const (
	StatusPending = "pending"
	StatusCreated = "created"
	StatusRunning = "running"
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

	// LogService is the configuration for the log service.
	//
	// +optional
	LogService *LogServiceConfig `json:"logService,omitempty"`

	// Operator is the configuration for the workflow run operator.
	//
	// +optional
	Operator *OperatorConfig `json:"operator,omitempty"`

	// MetadataAPI is the configuration for the step metadata-api server.
	//
	// +optional
	MetadataAPI *MetadataAPIConfig `json:"metadataAPI,omitempty"`

	// Vault is the configuration for accessing vault from the operator and metadata-api.
	//
	// +kubebuilder:default={sidecar: {image: "vault:latest"}}
	// +optional
	Vault *VaultConfig `json:"vault,omitempty"`

	// SentryDSNSecretName is the secret that holds the DSN address for Sentry
	// error and stacktrace collection. The secret object MUST have a data
	// field called "dsn".
	//
	// +optional
	SentryDSNSecretName *string `json:"sentryDSNSecretName,omitempty"`
}

// LogServiceConfig is the configuration for the relay-log-service deployment
type LogServiceConfig struct {
	// +kubebuilder:default="relaysh/relay-pls:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity is an optional set of affinity constraints to apply to operator
	// pods.
	//
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +optional
	VaultAgentRole *string `json:"vaultAgentRole,omitempty"`

	// CredentialsSecretName is the name of the secret containing the
	// credentials for the log service.
	//
	// +optional
	CredentialsSecretName string `json:"credentialsSecretName"`

	// Project is the BigQuery project to use for logging.
	//
	// +optional
	Project string `json:"project"`

	// Dataset is the BigQuery dataset to use for logging.
	//
	// +optional
	Dataset string `json:"dataset"`

	// Project is the BigQuery table to use for logging.
	//
	// +optional
	Table string `json:"table"`
}

// OperatorConfig is the configuration for the relay-operator deployment
type OperatorConfig struct {
	// +kubebuilder:default="relaysh/relay-operator:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// GenerateJWTSigningKey will generate a JWT signing key and store it in a
	// Secret for use by the operator pods. If this field is set to true, then
	// the below JWTSigningKeySecretName is ignored.
	//
	// +kubebuilder:default=false
	// +optional
	GenerateJWTSigningKey bool `json:"generateJWTSigningKey"`

	// JWTSigningKeySecretName is the name of the secret object that holds a
	// JWT signing key.  The secret object MUST have a data field called
	// "key.pem".  This field is ignored if GenerateJWTSigningKey is true.
	//
	// +optional
	JWTSigningKeySecretName *string `json:"jwtSigningKeySecretName,omitempty"`

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
	// +kubebuilder:default={image: "relaysh/relay-runtime-tools:latest"}
	ToolInjection *ToolInjectionConfig `json:"toolInjection,omitempty"`

	// AdmissionWebhookServer is the configuration for the
	// admissionregistration webhook server.  If this field is set, then the
	// admission webhook server is enabled and MutatingWebhooks are created.
	//
	// +optional
	AdmissionWebhookServer *AdmissionWebhookServerConfig `json:"admissionWebhookServer,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +optional
	VaultAgentRole *string `json:"vaultAgentRole,omitempty"`
}

type AdmissionWebhookServerConfig struct {
	// TLSSecretName is the name of the secret that holds the tls cert
	// files for webhooks. The secret object MUST have two data fields called
	// "tls.key" and "tls.crt".
	TLSSecretName string `json:"tlsSecretName,omitempty"`
	// CABundleSecretName is the name of the secret that holds the ca
	// certificate bundle for the admission webhook config.  The secret object
	// MUST have a field called "ca.crt".
	//
	// +optional
	CABundleSecretName *string `json:"caBundleSecretName,omitempty"`
}

// MetadataAPIConfig is the configuration for the relay-metadata-api deployment
type MetadataAPIConfig struct {
	// +kubebuilder:default="relaysh/relay-metadata-api:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

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

	// LogServiceEnabled defines whether the log service is enabled or not
	//
	// +optional
	LogServiceEnabled bool `json:"logServiceEnabled,omitempty"`

	// LogServiceURL is the URL of the service used to persist log messages.
	//
	// +optional
	LogServiceURL string `json:"logServiceURL,omitempty"`

	// StepMetadataURL is the URL to use to fetch step metadata for schema
	// validation.
	//
	// +kubebuilder:default="https://relay.sh/step-metadata.json"
	// +optional
	StepMetadataURL string `json:"stepMetadataURL,omitempty"`

	// VaultAgentRole is the role to use when configuring the vault agent.
	//
	// +optional
	VaultAgentRole *string `json:"vaultAgentRole,omitempty"`

	// +kubebuilder:default="tenant"
	// +optional
	VaultAuthRole string `json:"vaultAuthRole,omitempty"`

	// +kubebuilder:default="auth/jwt-tenants"
	// +optional
	VaultAuthPath string `json:"vaultAuthPath,omitempty"`
}

type VaultConfig struct {
	// +kubebuilder:default="pls"
	// +optional
	LogServicePath string `json:"logServicePath"`

	// +kubebuilder:default="metadata-api"
	// +optional
	TransitKey string `json:"transitKey"`

	// +kubebuilder:default="transit-tenants"
	// +optional
	TransitPath string `json:"transitPath"`

	// +kubebuilder:default="customers"
	// +optional
	TenantPath string `json:"tenantPath"`

	// Sidecar is the configuration for the vault sidecar containers used by
	// the operator and the metadata-api.
	//
	// +kubebuilder:default={image: "vault:latest"}
	// +optional
	Sidecar *VaultSidecar `json:"sidecar"`
}

type VaultSidecar struct {
	// +kubebuilder:default="vault:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`

	// Resources sets the resource requirements for the vault sidecar containers.
	//
	// +kubebuilder:default={limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// ServerAddr is the address to the vault server the sidecar agent should connect to.
	//
	// +kubebuilder:default="http://vault:8200"
	ServerAddr string `json:"serverAddr"`
}

type ToolInjectionConfig struct {
	// +kubebuilder:default="relaysh/relay-runtime-tools:latest"
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
}

// RelayCoreStatus defines the observed state of RelayCore
type RelayCoreStatus struct {
	// +optional
	Status Status `json:"status,omitempty"`
	// +optional
	LogServiceServiceAccount string `json:"logServiceServiceAccount,omitempty"`
	// +optional
	OperatorServiceAccount string `json:"operatorServiceAccount,omitempty"`
	// +optional
	MetadataAPIServiceAccount string `json:"metadataAPIServiceAccount,omitempty"`
	// +optional
	Vault VaultStatusSummary `json:"vault,omitempty"`
}

type VaultStatusSummary struct {
	// +optional
	JWTSigningKeySecret string `json:"jwtSigningKeySecret,omitempty"`
	// +optional
	LogServiceRole string `json:"logServiceRole,omitempty"`
	// +optional
	OperatorRole string `json:"operatorRole,omitempty"`
	// +optional
	MetadataAPIRole string `json:"metadataAPIRole,omitempty"`
	// +optional
	LogServiceServiceAccount string `json:"logServiceServiceAccount,omitempty"`
	// +optional
	OperatorServiceAccount string `json:"operatorServiceAccount,omitempty"`
	// +optional
	MetadataAPIServiceAccount string `json:"metadataAPIServiceAccount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type RelayCore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RelayCoreSpec `json:"spec"`

	// +optional
	Status RelayCoreStatus `json:"status,omitempty"`
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
