package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebhookTrigger represents a definition of a webhook to receive events.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
type WebhookTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WebhookTriggerSpec `json:"spec"`

	// +optional
	Status WebhookTriggerStatus `json:"status"`
}

type WebhookTriggerSpec struct {
	// TenantRef selects the tenant to apply this trigger to.
	TenantRef corev1.LocalObjectReference `json:"tenantRef"`

	// Image is the Docker image to run when this webhook receives an event.
	Image string `json:"image"`

	// Input is the input script to provide to the container.
	//
	// +optional
	Input []string `json:"input,omitempty"`

	// Command is the path to the executable to run when the container starts.
	//
	// +optional
	Command string `json:"command,omitempty"`

	// Args are the command arguments.
	//
	// +optional
	Args []string `json:"args,omitempty"`

	// Spec is the Relay specification to be provided to the container image.
	//
	// +optional
	Spec UnstructuredObject `json:"spec,omitempty"`
}

type WebhookTriggerStatus struct {
	// ObservedGeneration is the generation of the resource specification that
	// this status matches.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// URL is the endpoint for the webhook once provisioned.
	//
	// +optional
	URL string `json:"url,omitempty"`

	// Conditions are the observations of this resource's tate.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []WebhookTriggerCondition `json:"conditions,omitempty"`
}

type WebhookTriggerConditionType string

const (
	WebhookTriggerServiceReady WebhookTriggerConditionType = "ServiceReady"
	WebhookTriggerReady        WebhookTriggerConditionType = "Ready"
)

type WebhookTriggerCondition struct {
	Condition Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=ServiceReady;Ready
	Type WebhookTriggerConditionType `json:"type"`
}

// WebhookTriggerList enumerates many WebhookTrigger resources.
//
// +kubebuilder:object:root=true
type WebhookTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebhookTrigger `json:"items"`
}
