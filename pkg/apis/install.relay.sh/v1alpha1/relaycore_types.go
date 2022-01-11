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

	// JWTSigningKeys is the secret and keys that hold a JWT signing key pair
	// for the workflow run key signing operations with vault. This secret must
	// have 2 fields for a public and private key pair. If this field is not
	// set, then signings key will be generated when the relay core resources
	// are being persisted.
	//
	// +optional
	JWTSigningKeyRef *JWTSigningKeySource `json:"jwtSigningKeys,omitempty"`

	// LogService is the configuration for the log service.
	//
	// +kubebuilder:default={image: "relaysh/relay-pls:latest", imagePullPolicy: "IfNotPresent"}
	// +optional
	LogService LogServiceConfig `json:"logService,omitempty"`

	// Operator is the configuration for the workflow run operator.
	//
	// +optional
	Operator *OperatorConfig `json:"operator,omitempty"`

	// MetadataAPI is the configuration for the step metadata-api server.
	//
	// +kubebuilder:default={image: "relaysh/relay-metadata-api:latest", imagePullPolicy: "IfNotPresent"}
	// +optional
	MetadataAPI *MetadataAPIConfig `json:"metadataAPI,omitempty"`

	// Vault is the configuration for accessing vault from the operator and metadata-api.
	//
	// +kubebuilder:default={sidecar: {image: "vault:latest", imagePullPolicy: "IfNotPresent", resources: {limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}, serverAddr: "http://vault:8200"}}
	// +optional
	Vault *VaultConfig `json:"vault,omitempty"`

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
	Image string `json:"image"`

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
	CredentialsSecretKeyRef corev1.SecretKeySelector `json:"credentialsSecretKeyRef,omitempty"`

	// Project is the BigQuery project to use for logging.
	Project string `json:"project,omitempty"`

	// Dataset is the BigQuery dataset to use for logging.
	Dataset string `json:"dataset,omitempty"`

	// Project is the BigQuery table to use for logging.
	Table string `json:"table,omitempty"`
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
	// +kubebuilder:default="relaysh/relay-operator-webhook-certificate-controller:latest"
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

	// Auth provides credentials for vault server authentication.
	//
	// +optional
	Auth *VaultConfigAuth `json:"auth"`

	// AuthDelegatorServiceAccount is the name of the service account that
	// should be used to give vault token review access for the kubernetes auth
	// method.
	//
	// +optional
	AuthDelegatorServiceAccountName string `json:"authDelegatorServiceAccountName"`

	// ConfigMapRef is the reference to the config map that contains the
	// scripts and policies to configure vault with.
	//
	// +optional
	ConfigMapRef *VaultConfigMapSource `json:"configMapRef"`

	// Sidecar is the configuration for the vault sidecar containers used by
	// the operator and the metadata-api.
	//
	// +kubebuilder:default={image: "vault:latest", imagePullPolicy: "IfNotPresent", resources: {limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}, serverAddr: "http://vault:8200"}
	// +optional
	Sidecar *VaultSidecar `json:"sidecar"`
}

type VaultConfigMapSource struct {
	corev1.LocalObjectReference `json:",inline"`
}

type VaultConfigAuth struct {
	// Token is the token to use for vault server authentication when
	// configuring engine mounts and policies for relay-core components.
	//
	// +optional
	Token string `json:"token,omitempty"`

	// TokenFrom allows the vault server token to be provided by another source
	// such as a Secret.
	//
	// +optional
	TokenFrom *VaultTokenSource `json:"tokenFrom,omitempty"`

	// UnsealKey enables a Job to unseal a vault server. This is intended to
	// only be used by the development environment and must not be used in any
	// other environment. This Job only supports a singlular unseal key, so
	// servers that require multiple keys will not be unsealed.
	//
	// +optional
	UnsealKey string `json:"unsealKey,omitempty"`
}

type VaultTokenSource struct {
	// SecretKeyRef selects an API token by looking up the value in a secret.
	//
	// +optional
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
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
	// TriggerPoolName is the name of the tool injection pool for triggers.
	TriggerPoolName string `json:"triggerPoolName"`
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
