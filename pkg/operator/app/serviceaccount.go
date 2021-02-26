package app

import corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"

func ConfigureMetadataAPIServiceAccount(sa *corev1obj.ServiceAccount) {
	// This service account is used only for the metadata API to access cluster
	// resources using roles we set up in the target namespace.
	sa.Object.AutomountServiceAccountToken = func(b bool) *bool { return &b }(false)
}

func ConfigureUntrustedServiceAccount(sa *corev1obj.ServiceAccount) {
	// This is the default service account used for Tekton tasks and Knative
	// services. It has no permissions.
	sa.Object.AutomountServiceAccountToken = func(b bool) *bool { return &b }(false)
}
