package v1beta1

import corev1 "k8s.io/api/core/v1"

type Secret struct {
	// Name is a unique name for this secret.
	Name string `json:"name"`

	// Value is the hardcoded value of this secret.
	//
	// +optional
	Value *Unstructured `json:"value,omitempty"`

	// ValueFrom allows the value of this secret to be provided by another
	// resource.
	//
	// +optional
	ValueFrom *SecretSource `json:"valueFrom,omitempty"`

	// Patches adjusts the value of a secret after it is read from secure
	// storage. Each patch is applied in order.
	//
	// +optional
	Patches []*Patch `json:"patches,omitempty"`
}

type SecretSource struct {
	// SecretRef selects a secret value by combining all of the values in a
	// Kubernetes secret into an object.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// SecretKeyRef selects a secret value by looking up the value in a
	// particular key of a Kubernetes secret.
	//
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`

	// VaultPathRef selects a secret value by combining all of the values at a
	// path in Vault into an object.
	//
	// +optional
	VaultPathRef *VaultPathSelector `json:"vaultPathRef,omitempty"`

	// VaultFieldRef selects a secret value by looking up the secret in
	// HashiCorp Vault.
	//
	// +optional
	VaultFieldRef *VaultFieldSelector `json:"vaultFieldRef,omitempty"`

	// VaultFolderRef selects a secret by looking up a particular field in each
	// of the children found in a given folder in Vault.
	//
	// +optional
	VaultFolderRef *VaultFieldSelector `json:"vaultFolderRef,omitempty"`
}
