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

// RelayCoreSpec defines the desired state of RelayCore
type RelayCoreSpec struct {
	// +kubebuilder:default="dev"
	// +optional
	Environment string `json:"environment"`

	// +kubebuilder:default={image: "relaysh/relay-operator:latest"}
	// +optional
	Operator *OperatorConfig `json:"operator"`

	// +kubebuilder:default={image: "relaysh/relay-metadata-api:latest"}
	// +optional
	MetadataAPI *MetadataAPIConfig `json:"metadataAPI"`

	// +kubebuilder:default={sidecar: {image: "vault:latest"}}
	// +optional
	Vault *VaultConfig `json:"vault"`
}

// OperatorConfig is the configuration for the relay-operator deployment
type OperatorConfig struct {
	// StorageAddr is the storage address URI for log storage.
	//
	// +optional
	StorageAddr string `json:"storageAddr"`

	// +kubebuilder:default="relaysh/relay-operator:latest"
	//
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	//
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
	JWTSigningKeySecretName *string `json:"jwtSigningKeySecretName"`

	// MetricsEnabled enables the metrics server for the operator deployment
	// and creates a service that can be used to scrape those metrics.
	//
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
	NodeSelector map[string]string `json:"nodeSelector"`

	// SentryDSNSecretName is the secret that holds the DSN address for Sentry
	// error and stacktrace collection. The secret object MUST have a data
	// field called "dsn".
	//
	// +optional
	SentryDSNSecretName *string `json:"sentryDSNSecretName,omitempty"`

	// +kubebuilder:default=false
	// +optional
	Standalone bool `json:"standalone"`

	// Workers is the number of workers the operator should run to process workflows
	//
	// +kubebuilder:default=2
	// +optional
	Workers int32 `json:"workers,omitempty"`

	// ToolInjection is the configuration for the entrypointer and tool injection runtime tooling.
	//
	// +kubebuilder:default={image: "relaysh/relay-runtime-tools:latest"}
	ToolInjection *ToolInjectionConfig `json:"toolInjection,omitempty"`

	// WebhookTLSSecretName is the name of the secret that holds the tls cert files for webhooks.
	// The secret object MUST have two data fields called "tls.key" and "tls.crt".
	//
	// +optional
	WebhookTLSSecretName *string `json:"webhookTLSSecretName"`
}

// MetadataAPIConfig is the configuration for the relay-metadata-api deployment
type MetadataAPIConfig struct {
	// +kubebuilder:default="relaysh/relay-metadata-api:latest"
	// +optional
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`

	// +optional
	Env []corev1.EnvVar `json:"env"`

	// +optional
	NodeSelector map[string]string `json:"nodeSelector"`

	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas"`

	// SentryDSNSecretName is the secret that holds the DSN address for Sentry
	// error and stacktrace collection. The secret object MUST have a data
	// field called "dsn".
	//
	// +optional
	SentryDSNSecretName *string `json:"sentryDSNSecretName,omitempty"`

	// +optional
	TLSSecretName *string `json:"tlsSecretName"`

	// +optional
	URL *string `json:"url,omitempty"`
}

type VaultConfig struct {
	// +kubebuilder:default="metadata-api"
	// +optional
	TransitKey string `json:"transitKey"`

	// +kubebuilder:default="transit-tenants"
	// +optional
	TransitPath string `json:"transitPath"`

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

	// +kubebuilder:default={limits: {cpu: "50m", memory: "64Mi"}, requests: {cpu: "25m", memory: "32Mi"}}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// +kubebuilder:default={mountPath: "/var/run/vault/config", name: "vault-agent-conf", readOnly: true}
	// +optional
	ConfigVolumeMounts corev1.VolumeMount `json:"configVolumeMount"`

	// +kubebuilder:default={mountPath: "/var/run/secrets/kubernetes.io/serviceaccount@vault", name: "vault-agent-sa-token", readOnly: true}
	// +optional
	ServiceAccountVolumeMount corev1.VolumeMount `json:"serviceAccountVolumeMount"`

	// ServerAddr is the address to the vault server the sidecar agent should connect to.
	//
	// +kubebuilder:default="http://vault:8200"
	ServerAddr string `json:"serverAddr"`
}

type ToolInjectionConfig struct {
	Image string `json:"image"`

	// +kubebuilder:default="IfNotPresent"
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
}

// RelayCoreStatus defines the observed state of RelayCore
type RelayCoreStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// RelayCore is the Schema for the relaycores API
type RelayCore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RelayCoreSpec   `json:"spec,omitempty"`
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
