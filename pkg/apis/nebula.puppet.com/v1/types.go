// +groupName=nebula.puppet.com
package v1

import (
	"encoding/json"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// WorkflowRun is the root type for a workflow run.
//
// +kubebuilder:object:root=true
type WorkflowRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowRunSpec `json:"spec"`

	// +optional
	State  WorkflowRunState  `json:"state,omitempty"`
	Status WorkflowRunStatus `json:"status,omitempty"`
}

type WorkflowRunParameters UnstructuredObject

type WorkflowRunSpec struct {
	Name       string                `json:"name"`
	Parameters WorkflowRunParameters `json:"parameters,omitempty"`
	Workflow   Workflow              `json:"workflow,omitempty"`
}

type WorkflowParameters UnstructuredObject

type Workflow struct {
	Name       string             `json:"name"`
	Parameters WorkflowParameters `json:"parameters,omitempty"`
	Steps      []*WorkflowStep    `json:"steps"`
}

type WorkflowStep struct {
	Name      string             `json:"name"`
	Image     string             `json:"image,omitempty"`
	Spec      UnstructuredObject `json:"spec,omitempty"`
	Input     []string           `json:"input,omitempty"`
	Command   string             `json:"command,omitempty"`
	Args      []string           `json:"args,omitempty"`
	When      Unstructured       `json:"when,omitempty"`
	DependsOn []string           `json:"depends_on,omitempty"`
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

type WorkflowState UnstructuredObject

type WorkflowRunState struct {
	Workflow WorkflowState            `json:"workflow"`
	Steps    map[string]WorkflowState `json:"steps"`
}

// +kubebuilder:object:root=true
type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}

// +kubebuilder:validation:Type=object
type Unstructured struct {
	value transfer.JSONInterface `json:"-"`
}

func (u Unstructured) Value() interface{} {
	return u.value.Data
}

func (u Unstructured) MarshalJSON() ([]byte, error) {
	return u.value.MarshalJSON()
}

func (u *Unstructured) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &u.value)
}

func (in *Unstructured) DeepCopy() *Unstructured {
	if in == nil {
		return nil
	}

	out := &Unstructured{}
	*out = *in
	out.value = transfer.JSONInterface{
		Data: runtime.DeepCopyJSONValue(in.value.Data),
	}
	return out
}

func AsUnstructured(value interface{}) Unstructured {
	return Unstructured{
		value: transfer.JSONInterface{Data: value},
	}
}

type UnstructuredObject map[string]Unstructured

func (uo UnstructuredObject) Value() map[string]interface{} {
	out := make(map[string]interface{}, len(uo))
	for k, v := range uo {
		out[k] = v.Value()
	}
	return out
}

func NewUnstructuredObject(value map[string]interface{}) UnstructuredObject {
	out := make(map[string]Unstructured, len(value))
	for k, v := range value {
		out[k] = AsUnstructured(v)
	}
	return out
}
