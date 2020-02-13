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
	State  WorkflowRunState  `json:"state,omitempty"`
	Status WorkflowRunStatus `json:"status,omitempty"`
}

type WorkflowRunSpec struct {
	Name       string                `json:"name"`
	Parameters WorkflowRunParameters `json:"parameters,omitempty"`
	Workflow   Workflow              `json:"workflow,omitempty"`
}

type Workflow struct {
	Name       string             `json:"name"`
	Parameters WorkflowParameters `json:"parameters,omitempty"`
	Steps      []*WorkflowStep    `json:"steps"`
}

type WorkflowCondition struct {
	Name string
	Type string
}

type WorkflowStep struct {
	Name       string              `json:"name"`
	Image      string              `json:"image,omitempty"`
	Spec       WorkflowStepSpec    `json:"spec,omitempty"`
	Input      []string            `json:"input,omitempty"`
	Command    string              `json:"command,omitempty"`
	Args       []string            `json:"args,omitempty"`
	When       WorkflowWhen        `json:"when,omitempty"`
	Conditions []WorkflowCondition `json:"conditions,omitempty"`
	DependsOn  []string            `json:"depends_on,omitempty"`
}

type WorkflowRunStatusSummary struct {
	Name           string       `json:"name"`
	Status         string       `json:"status"`
	StartTime      *metav1.Time `json:"startTime"`
	CompletionTime *metav1.Time `json:"completionTime"`
}

type WorkflowRunStatus struct {
	Status         string                              `json:"status"`
	StartTime      *metav1.Time                        `json:"startTime"`
	CompletionTime *metav1.Time                        `json:"completionTime"`
	Steps          map[string]WorkflowRunStatusSummary `json:"steps"`
	Conditions     map[string]WorkflowRunStatusSummary `json:"conditions"`
}

type WorkflowRunState struct {
	Workflow WorkflowState            `json:"workflow"`
	Steps    map[string]WorkflowState `json:"steps"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}

type WorkflowState map[string]interface{}

type WorkflowParameters map[string]interface{}

type WorkflowRunParameters map[string]interface{}

type WorkflowStepSpec map[string]interface{}

type WorkflowWhen []interface{}

func (in *WorkflowParameters) DeepCopy() *WorkflowParameters {
	if in == nil {
		return nil
	}

	out := make(WorkflowParameters)
	for key, value := range *in {
		out[key] = value
	}

	return &out
}

func (in *WorkflowRunParameters) DeepCopy() *WorkflowRunParameters {
	if in == nil {
		return nil
	}

	out := make(WorkflowRunParameters)
	for key, value := range *in {
		out[key] = value
	}

	return &out
}

func (in *WorkflowStepSpec) DeepCopy() *WorkflowStepSpec {
	if in == nil {
		return nil
	}

	out := make(WorkflowStepSpec)
	for key, value := range *in {
		out[key] = value
	}

	return &out
}

func (in *WorkflowState) DeepCopy() *WorkflowState {
	if in == nil {
		return nil
	}

	out := make(WorkflowState)
	for key, value := range *in {
		out[key] = value
	}

	return &out
}

func (in *WorkflowWhen) DeepCopy() *WorkflowWhen {
	if in == nil {
		return nil
	}

	out := make(WorkflowWhen, len(*in))
	for index, value := range *in {
		out[index] = value
	}

	return &out
}
