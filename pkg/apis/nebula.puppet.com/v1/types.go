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

	Spec   SecretAuthSpec   `json:"spec"`
	Status SecretAuthStatus `json:"status"`
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
	// TODO this needs to eventually implement the k8s conditions api,
	// but given how much work that is, we will just accept an empty string
	// as not ready and Ready as it is ready.
	Ready string `json:"ready"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +groupName=nebula.puppet.com

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
	Name     string   `json:"name"`
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
