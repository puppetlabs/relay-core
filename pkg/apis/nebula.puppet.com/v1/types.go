package v1

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowRun is the root type for a workflow run.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
type WorkflowRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowRunSpec `json:"spec"`

	// +optional
	State WorkflowRunState `json:"state,omitempty"`

	// +optional
	Status WorkflowRunStatus `json:"status,omitempty"`
}

type WorkflowRunSpec struct {
	Name     string   `json:"name"`
	Workflow Workflow `json:"workflow"`

	// +optional
	Parameters relayv1beta1.UnstructuredObject `json:"parameters,omitempty"`

	// +optional
	TenantRef *corev1.LocalObjectReference `json:"tenantRef,omitempty"`
}

type Workflow struct {
	Name  string          `json:"name"`
	Steps []*WorkflowStep `json:"steps"`

	// +optional
	Parameters relayv1beta1.UnstructuredObject `json:"parameters,omitempty"`
}

type WorkflowStep struct {
	Name string `json:"name"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Spec relayv1beta1.UnstructuredObject `json:"spec,omitempty"`

	// +optional
	Input []string `json:"input,omitempty"`

	// +optional
	Command string `json:"command,omitempty"`

	// +optional
	Args []string `json:"args,omitempty"`

	// +optional
	Env relayv1beta1.UnstructuredObject `json:"env,omitempty"`

	// +optional
	When relayv1beta1.Unstructured `json:"when,omitempty"`

	// +optional
	DependsOn []string `json:"depends_on,omitempty"`
}

type WorkflowRunStatusSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`

	// +optional
	LogKey string `json:"logKey,omitempty"`

	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

type WorkflowRunStatus struct {
	Status string `json:"status"`

	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// +optional
	Steps map[string]WorkflowRunStatusSummary `json:"steps,omitempty"`

	// +optional
	Conditions map[string]WorkflowRunStatusSummary `json:"conditions,omitempty"`
}

type WorkflowRunState struct {
	// +optional
	Workflow relayv1beta1.UnstructuredObject `json:"workflow,omitempty"`

	// +optional
	Steps map[string]relayv1beta1.UnstructuredObject `json:"steps,omitempty"`
}

// WorkflowRunList enumerates many WorkflowRun resources.
//
// +kubebuilder:object:root=true
type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}
