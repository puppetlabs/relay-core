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
	// WorkflowID is used to build paths to secrets in vault.
	WorkflowID string `json:"workflowID"`
	// WorkflowRunID is used to namespace objects when they are created.
	// It's more than likely that this will be the same as the k8s namespace,
	// but still required as they can be different.
	WorkflowRunID string `json:"workflowRunID"`
}

type SecretAuthStatus struct {
	MetadataServicePod             string `json:"metadataServicePod"`
	MetadataServiceService         string `json:"metadataServiceService"`
	MetadataServiceImagePullSecret string `json:"metadataServiceImagePullSecret"`
	ServiceAccount                 string `json:"serviceAccount"`
	Role                           string `json:"Role"`
	RoleBinding                    string `json:"RoleBinding"`
	ConfigMap                      string `json:"configMap"`
	VaultPolicy                    string `json:"vaultPolicy"`
	VaultAuthRole                  string `json:"vaultAuthRole"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecretAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretAuth `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +groupName=nebula.puppet.com

type WorkflowRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowRunSpec `json:"spec"`

	// +optional
	Status WorkflowRunStatus `json:"status,omitempty"`
}

type WorkflowRunSpec struct {
	Name string `json:"name"`

	Workflow Workflow `json:"workflow,omitempty"`
}

type Workflow struct {
	Name string `json:"name"`
}

type WorkflowRunStep struct {
	Name           string       `json:"name"`
	Status         string       `json:"status"`
	StartTime      *metav1.Time `json:"startTime"`
	CompletionTime *metav1.Time `json:"completionTime"`
}

type WorkflowRunStatus struct {
	Status         string                     `json:"status"`
	StartTime      *metav1.Time               `json:"startTime"`
	CompletionTime *metav1.Time               `json:"completionTime"`
	Steps          map[string]WorkflowRunStep `json:"steps"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}
