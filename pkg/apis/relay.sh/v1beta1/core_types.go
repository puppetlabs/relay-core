package v1beta1

import corev1 "k8s.io/api/core/v1"

type SecretKeySelector struct {
	corev1.LocalObjectReference `json:",inline"`

	// Key is the key from the secret to use.
	Key string `json:"key"`
}
