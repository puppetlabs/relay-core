package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition is the base type for Relay conditions, inlined into each condition
// type.
type Condition struct {
	Status             corev1.ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime"`

	// Reason identifies the cause of the given status using an API-locked
	// camel-case identifier.
	//
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable description of the given status.
	//
	// +optional
	Message string `json:"message,omitempty"`
}
