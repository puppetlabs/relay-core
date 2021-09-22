package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{Group: "relay.sh", Version: "v1beta1"}

func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme

	TenantKind         = SchemeGroupVersion.WithKind("Tenant")
	WebhookTriggerKind = SchemeGroupVersion.WithKind("WebhookTrigger")
	WorkflowKind       = SchemeGroupVersion.WithKind("Workflow")
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Tenant{},
		&TenantList{},
		&WebhookTrigger{},
		&WebhookTriggerList{},
		&Workflow{},
		&WorkflowList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
