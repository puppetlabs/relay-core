package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Workflow Workflow `json:"workflow,omitempty"`
}

type Workflow struct {
	ID   string `json:"id"`
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
