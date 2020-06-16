package obj

import (
	"context"
	"errors"
	"time"

	"github.com/puppetlabs/relay-core/pkg/util/retry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceAccountDefaultTokenSecretLoadTimeout          = 120 * time.Second
	ServiceAccountDefaultTokenSecretLoadBackoffFrequency = 250 * time.Millisecond
)

var (
	ErrServiceAccountMissingDefaultTokenSecret = errors.New("obj: service account has no default token secret")
)

type ServiceAccount struct {
	Key    client.ObjectKey
	Object *corev1.ServiceAccount

	DefaultTokenSecret *ServiceAccountTokenSecret
}

var _ Persister = &ServiceAccount{}
var _ Loader = &ServiceAccount{}
var _ Ownable = &ServiceAccount{}
var _ LabelAnnotatableFrom = &ServiceAccount{}

func (sa *ServiceAccount) Persist(ctx context.Context, cl client.Client) error {
	if err := CreateOrUpdate(ctx, cl, sa.Key, sa.Object); err != nil {
		return err
	}

	_, err := RequiredLoader{sa}.Load(ctx, cl)
	return err
}

func (sa *ServiceAccount) Load(ctx context.Context, cl client.Client) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, ServiceAccountDefaultTokenSecretLoadTimeout)
	defer cancel()

	var ok bool
	err := retry.Retry(ctx, ServiceAccountDefaultTokenSecretLoadBackoffFrequency, func() *retry.RetryError {
		if ok, err := GetIgnoreNotFound(ctx, cl, sa.Key, sa.Object); err != nil || !ok {
			return retry.RetryPermanent(err)
		}

		if err := sa.loadDefaultTokenSecret(ctx, cl); err != nil {
			return retry.RetryTransient(err)
		}

		ok = true
		return retry.RetryPermanent(nil)
	})
	return ok, err
}

func (sa *ServiceAccount) loadDefaultTokenSecret(ctx context.Context, cl client.Client) error {
	if len(sa.Object.Secrets) == 0 || sa.Object.Secrets[0].Name == "" {
		return ErrServiceAccountMissingDefaultTokenSecret
	}

	key := client.ObjectKey{
		Namespace: sa.Key.Namespace,
		Name:      sa.Object.Secrets[0].Name,
	}
	if sa.DefaultTokenSecret == nil {
		sa.DefaultTokenSecret = NewServiceAccountTokenSecret(key)
	}

	if sa.DefaultTokenSecret.Key == key && len(sa.DefaultTokenSecret.Object.GetUID()) != 0 {
		// Already loaded.
		return nil
	}

	if _, err := (RequiredLoader{sa.DefaultTokenSecret}).Load(ctx, cl); err != nil {
		return err
	}

	return nil
}

func (sa *ServiceAccount) Owned(ctx context.Context, owner Owner) error {
	return Own(sa.Object, owner)
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

func ConfigureUntrustedServiceAccount(sa *ServiceAccount) {
	// This is the default service account used for Tekton tasks and Knative
	// services. It has no permissions.
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
