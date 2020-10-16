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
	Operator    OperatorConfig    `json:"operator"`
	MetadataAPI MetadataAPIConfig `json:"metadataAPI"`
}

// OperatorConfig is the configuration for the relay-operator deployment
type OperatorConfig struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
	Env             []corev1.EnvVar   `json:"env"`
	NodeSelector    map[string]string `json:"nodeSelector"`

	// +optional
	Standalone *bool `json:"standalone,omitempty"`

	StorageAddr  string       `json:"storageAddr"`
	VaultSidecar VaultSidecar `json:"vaultSidecar"`
	Workers      int32        `json:"workers"`
}

// MetadataAPIConfig is the configuration for the relay-metadata-api deployment
type MetadataAPIConfig struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
	Env             []corev1.EnvVar   `json:"env"`
	NodeSelector    map[string]string `json:"nodeSelector"`
	VaultSidecar    VaultSidecar      `json:"vaultSidecar"`
}

type VaultSidecar struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy"`
}

// RelayCoreStatus defines the observed state of RelayCore
type RelayCoreStatus struct {
}

// +kubebuilder:object:root=true

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
