package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceAccount struct {
	Key    client.ObjectKey
	Object *corev1.ServiceAccount
}

var _ Persister = &ServiceAccount{}
var _ Loader = &ServiceAccount{}
var _ Ownable = &ServiceAccount{}
var _ LabelAnnotatableFrom = &ServiceAccount{}

func (sa *ServiceAccount) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, sa.Key, sa.Object)
}

func (sa *ServiceAccount) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, sa.Key, sa.Object)
}

func (sa *ServiceAccount) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&sa.Object.ObjectMeta, ref)
}

func (sa *ServiceAccount) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&sa.Object.ObjectMeta, from)
}

func NewServiceAccount(key client.ObjectKey) *ServiceAccount {
	return &ServiceAccount{
		Key:    key,
		Object: &corev1.ServiceAccount{},
	}
}

func ConfigureMetadataAPIServiceAccount(sa *ServiceAccount) {
	// This service account is used only for the metadata API to access cluster
	// resources using roles we set up in the target namespace.
	sa.Object.AutomountServiceAccountToken = func(b bool) *bool { return &b }(false)
}

func ConfigurePipelineServiceAccount(sa *ServiceAccount) {
	// This is the default service account used for Tekton tasks. It has no
	// permissions.
	sa.Object.AutomountServiceAccountToken = func(b bool) *bool { return &b }(false)
}

type systemServiceAccountOptions struct {
	imagePullSecrets []corev1.LocalObjectReference
}

type SystemServiceAccountOption func(opts *systemServiceAccountOptions)

func SystemServiceAccountWithImagePullSecret(ref corev1.LocalObjectReference) SystemServiceAccountOption {
	return func(opts *systemServiceAccountOptions) {
		opts.imagePullSecrets = append(opts.imagePullSecrets, ref)
	}
}

func ConfigureSystemServiceAccount(sa *ServiceAccount, opts ...SystemServiceAccountOption) {
	// This service account is used by internal containers needing to pull
	// restricted images.

	sao := &systemServiceAccountOptions{}

	for _, opt := range opts {
		opt(sao)
	}

	sa.Object.AutomountServiceAccountToken = func(b bool) *bool { return &b }(false)
	sa.Object.ImagePullSecrets = sao.imagePullSecrets
}
