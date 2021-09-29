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
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
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
	Name string `json:"name"`

	WorkflowRef corev1.LocalObjectReference `json:"workflowRef"`

	// +optional
	Parameters relayv1beta1.UnstructuredObject `json:"parameters,omitempty"`

	// +optional
	TenantRef *corev1.LocalObjectReference `json:"tenantRef,omitempty"`

	// WorkflowExecutionSink represents the destrination for workflow run requests.
	// If not specified, the metadata-api workflow run endpoint will reject a
	// request to run a workflow.
	//
	// +optional
	WorkflowExecutionSink *relayv1beta1.WorkflowExecutionSink `json:"workflowExecutionSink,omitempty"`
}

type WorkflowRunStatusSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`

	// +optional
	Outputs relayv1beta1.UnstructuredObject `json:"outputs,omitempty"`

	// +optional
	LogKey string `json:"logKey,omitempty"`

	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// +optional
	InitTime *metav1.Time `json:"initTime,omitempty"`

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
