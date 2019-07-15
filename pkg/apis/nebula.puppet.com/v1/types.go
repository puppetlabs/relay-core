package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +groupName=nebula.puppet.com

type SecretAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SecretAuthSpec `json:"spec"`

	// +optional
	Status SecretAuthStatus `json:"status,omitempty"`
}

type SecretAuthSpec struct {
	WorkflowID string `json:"workflowID"`
	RunNum     int    `json:"runNum"`
}

type SecretAuthStatus struct {
	MetadataServicePod     string `json:"metadataServicePod"`
	MetadataServiceService string `json:"metadataServiceService"`
	ServiceAccount         string `json:"serviceAccount"`
	ClusterRoleBinding     string `json:"clusterRoleBinding"`
	ConfigMap              string `json:"configMap"`
	VaultPolicy            string `json:"vaultPolicy"`
	VaultAuthRole          string `json:"vaultAuthRole"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecretAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretAuth `json:"items"`
}
