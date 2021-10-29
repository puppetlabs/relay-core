package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Log struct {
	// Name of the associated log.
	//
	// +optional
	Name string `json:"name"`

	// Context of the associated log.
	//
	// +optional
	Context string `json:"context"`
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
	// StepCompleted indicates whether a step has finished executing.
	StepCompleted StepConditionType = "Completed"

	// StepSkipped indicates a step has been skipped.
	StepSkipped StepConditionType = "Skipped"

	// StepSucceeded indicates a step has succeeded.
	StepSucceeded StepConditionType = "Succeeded"
)

type StepCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=Completed;Skipped;Succeeded
	Type StepConditionType `json:"type"`
}

type StepStatus struct {
	// Name is the name of this step.
	Name string `json:"name"`

	// Outputs are each of the outputs provided by this step, if available.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	Outputs []*StepOutput `json:"outputs,omitempty"`

	// +optional
	// +listType=map
	// +listMapKey=name
	Decorators []*Decorator `json:"decorators,omitempty"`

	// StartTime is the time this step began executing.
	//
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time this step ended, whether successful or not.
	//
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// InitializationTime is the time taken to initialize the step.
	//
	// +optional
	InitializationTime *metav1.Time `json:"initTime,omitempty"`

	// Associated logs for this step.
	//
	// +optional
	Logs []*Log `json:"logs,omitempty"`

	// Conditions are the possible observable conditions for this step.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []StepCondition `json:"conditions,omitempty"`
}
