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
	// ParameterValues assigns values to parameters defined in the workflow.
	//
	// +optional
	ParameterValues UnstructuredObject `json:"parameterValues,omitempty"`

	// WorkflowRef selects an already defined workflow to use for this run. Only
	// one of WorkflowRef, Workflow, or Step may be specified.
	//
	// +optional
	WorkflowRef *corev1.LocalObjectReference `json:"workflowRef,omitempty"`

	// Workflow allows workflow content to be specified inline as part of the
	// run. Only one of WorkflowRef, Workflow, or Step may be specified.
	//
	// +optional
	Workflow *WorkflowSpec `json:"workflow,omitempty"`

	// Step causes this run to execute only the given inlined step as a
	// standalone request.
	//
	// +optional
	Step *StepSpec `json:"step,omitempty"`
}

type StepSpec struct {
	Container `json:",inline"`

	// TenantRef selects the tenant to use for this step.
	TenantRef corev1.LocalObjectReference `json:"tenantRef"`

	// Parameters are the definitions of parameters used by this step's spec.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Parameters []*Parameter `json:"parameters,omitempty"`

	// Secrets are the definitions of secret data used by this step's spec.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Secrets []*Secret `json:"secrets,omitempty"`
}

type RunConditionType string

const (
	// RunCompleted indicates whether an entire run has finished executing. The
	// following tuples of (status, reason) are possible for this condition:
	//
	// - (Unknown, RunInitializing)
	// - (Unknown, RunInProgress)
	// - (False, RunSystemError)
	// - (True, RunSuccess)
	// - (True, RunFailure)
	// - (True, RunCancelled)
	//
	// These correspond to the definition of resolved and unresolved (for the
	// status) and the corresponding outcome (for the reason) in RFC 5, but are
	// broadly compatible with other Kubernetes conditions.
	RunCompleted RunConditionType = "Completed"
)

type RunCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=Completed
	Type RunConditionType `json:"type"`
}

type RunStatus struct {
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

type StepOutput struct {
	// Name is the name of this output.
	Name string `json:"name"`

	// Sensitive is whether this output contains sensitive or privileged data.
	// If this output is sensitive, the value will not be set.
	//
	// +optional
	Sensitive bool `json:"sensitive"`

	// Value is the value provided by the step for the output.
	//
	// +optional
	Value *Unstructured `json:"value"`
}

type StepConditionType string

const (
	// StepCompleted indicates whether a step has finished executing. The
	// following tuples of (status, reason) are possible for this condition:
	//
	// - (Unknown, StepInitializing)
	// - (Unknown, StepPending)
	// - (Unknown, StepInProgress)
	// - (False, StepSystemError)
	// - (False, StepSkipped)
	// - (True, StepSuccess)
	// - (True, StepFailure)
	// - (True, StepCancelled)
	//
	// These correspond to the definition of resolved and unresolved (for the
	// status) and the corresponding step status (for the reason) in RFC 5, but
	// are broadly compatible with other Kubernetes conditions.
	StepCompleted StepConditionType = "Completed"
)

type StepCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=Completed
	Type StepConditionType `json:"type"`
}

type StepStatus struct {
	// Name is the name of this step, or an empty string if this run is a
	// one-off step run.
	//
	// +optional
	Name string `json:"name,omitempty"`

	// Spec is the fully resolved spec for this step, if available. A spec value
	// provided by a secret will remain unresolved in this representation.
	//
	// +optional
	Spec UnstructuredObject `json:"spec,omitempty"`

	// Outputs are each of the outputs provided by this step, if available.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Outputs []*StepOutput `json:"outputs,omitempty"`

	// StartTime is the time this step began executing.
	//
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time this step ended, whether successful or not.
	//
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Conditions are the possible observable conditions for this step.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []StepCondition `json:"conditions,omitempty"`
}

// RunList enumerates many Run resources.
//
// +kubebuilder:object:root=true
type RunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Run `json:"items"`
}
