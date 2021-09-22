package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Workflow represents a set of steps that Relay can execute.
//
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowSpec `json:"spec"`
}

type WorkflowSpec struct {
	// TenantRef selects the tenant to use for this workflow.
	TenantRef corev1.LocalObjectReference `json:"tenantRef"`

	// Parameters are the definitions of parameters used by this workflow.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Parameters []*Parameter `json:"parameters,omitempty"`

	// Steps are the individual steps that make up the workflow.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Steps []*Step `json:"steps,omitempty"`
}

type Parameter struct {
	// Name is a unique name for this parameter.
	Name string `json:"name"`

	// Value is the default value for this parameter. If not specified, a value
	// must be provided at runtime.
	//
	// +optional
	Value *Unstructured `json:"default,omitempty"`
}

type Step struct {
	// Name is a unique name for this step.
	Name string `json:"name"`

	// Container defines the properties of the Docker container to run.
	Container `json:",inline"`

	// When provides a set of conditions that must be met for this step to run.
	//
	// +optional
	When *Unstructured `json:"when,omitempty"`

	// DependsOn causes this step to run after the given step names.
	//
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`
}

// WorkflowList enumerates many Workflow resources.
//
// +kubebuilder:object:root=true
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
