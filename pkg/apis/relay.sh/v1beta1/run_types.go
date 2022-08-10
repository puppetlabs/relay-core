package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Run is a request to invoke a workflow.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
type Run struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RunSpec `json:"spec"`

	// +optional
	Status RunStatus `json:"status,omitempty"`
}

type RunSpec struct {
	// Parameters assigns values to parameters defined in the workflow.
	//
	// +optional
	Parameters UnstructuredObject `json:"parameters,omitempty"`

	// State allows applying desired state changes.
	//
	// +optional
	State RunState `json:"state,omitempty"`

	// WorkflowRef selects a defined workflow to use for this run.
	WorkflowRef corev1.LocalObjectReference `json:"workflowRef"`
}

type RunConditionType string

const (
	// RunCancelled indicates whether a run was cancelled.
	RunCancelled RunConditionType = "Cancelled"

	// RunCompleted indicates whether an entire run has finished executing.
	RunCompleted RunConditionType = "Completed"

	// RunSucceeded indicates a run has succeeded.
	RunSucceeded RunConditionType = "Succeeded"
)

type RunCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=Cancelled;Completed;Succeeded
	Type RunConditionType `json:"type"`
}

type RunState struct {
	// Workflow allows applying desired workflow state changes.
	//
	// +optional
	Workflow UnstructuredObject `json:"workflow,omitempty"`

	// Step allows applying desired step state changes.
	// +optional
	Steps map[string]UnstructuredObject `json:"steps,omitempty"`
}

type RunStatus struct {
	// ObservedGeneration is the generation of the resource specification that
	// this status matches.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Steps provides information about the status of each step that makes up
	// this workflow run.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Steps []*StepStatus `json:"steps,omitempty"`

	// StartTime is the this run began executing.
	//
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time this run ended, whether successful or not.
	//
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Conditions are the possible observable conditions for this run.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []RunCondition `json:"conditions,omitempty"`
}

// RunList enumerates many Run resources.
//
// +kubebuilder:object:root=true
type RunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Run `json:"items"`
}
