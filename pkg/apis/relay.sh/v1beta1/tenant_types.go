package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tenant represents a scoping mechanism for runs and triggers.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TenantSpec `json:"spec"`

	// +optional
	Status TenantStatus `json:"status"`
}

type TenantSpec struct {
	// NamespaceTemplate defines a template for a namespace that will be created
	// for this scope. If not specified, resources are created in the namespace
	// of this resource.
	//
	// +optional
	NamespaceTemplate NamespaceTemplate `json:"namespaceTemplate,omitempty"`

	// +optional
	ToolInjection ToolInjection `json:"toolInjection,omitempty"`

	// TriggerEventSink represents the destination for events received as part
	// of trigger processing. If not specified, events will be logged and
	// discarded.
	//
	// +optional
	TriggerEventSink TriggerEventSink `json:"triggerEventSink,omitempty"`
}

type NamespaceTemplate struct {
	// Metadata is the metadata to associate with the namespace to create, such
	// as a name and list of labels. If not specified, values are automatically
	// generated.
	//
	// Labels from the tenant are automatically propagated onto the created
	// namespace.
	//
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
}

type ToolInjection struct {
	// VolumeClaimTemplate is an optional definition of the PVC that will be
	// populated and attached to every tenant container.
	//
	// +optional
	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

// TriggerEventSink represents the destination for trigger events. At most one
// of the fields may be specified at any one given time. If more than one is
// specified, the behavior is undefined.
type TriggerEventSink struct {
	// API is an event sink for the propretiary Relay API.
	//
	// +optional
	API *APITriggerEventSink `json:"api,omitempty"`
}

type APITriggerEventSink struct {
	URL string `json:"url"`

	// Token is the API token to use.
	//
	// +optional
	Token string `json:"token,omitempty"`

	// TokenFrom allows the API token to be provided by another resource.
	//
	// +optional
	TokenFrom *APITokenSource `json:"tokenFrom,omitempty"`
}

type APITokenSource struct {
	// SecretKeyRef selects an API token by looking up the value in a secret.
	//
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

type SecretKeySelector struct {
	corev1.LocalObjectReference `json:",inline"`

	// Key is the key from the secret to use.
	Key string `json:"key"`
}

type TenantStatus struct {
	// ObservedGeneration is the generation of the resource specification that
	// this status matches.
	//
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Namespace is the namespace managed by this tenant or the namespace of the
	// tenant if it is unmanaged.
	//
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Conditions are the observations of this resource's state.
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []TenantCondition `json:"conditions,omitempty"`
}

type TenantConditionType string

const (
	// TenantNamespaceReady indicates whether the namespace requested by the
	// tenant namespace template is ready to use.
	TenantNamespaceReady TenantConditionType = "NamespaceReady"

	// TenantEventSinkReady indicates whether the event sink can be used. For
	// example, any secret references must be resolvable.
	TenantEventSinkReady TenantConditionType = "EventSinkReady"

	// TenantVolumeClaimReady indicates whether the volume claim requested by the
	// tenant volume claim template is ready to use.
	TenantVolumeClaimReady TenantConditionType = "VolumeClaimReady"

	// TenantReady is set when all other conditions are ready.
	TenantReady TenantConditionType = "Ready"
)

type TenantCondition struct {
	Condition `json:",inline"`

	// Type is the identifier for this condition.
	//
	// +kubebuilder:validation:Enum=NamespaceReady;EventSinkReady;VolumeClaimReady;Ready
	Type TenantConditionType `json:"type"`
}

// TenantList enumerates many Tenant resources.
//
// +kubebuilder:object:root=true
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}
