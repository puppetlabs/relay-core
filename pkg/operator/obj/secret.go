package obj

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrNotOpaqueSecret                = errors.New("obj: secret is not an unstructured opaque secret")
	ErrNotImagePullSecret             = errors.New("obj: secret is not usable for pulling container images")
	ErrNotServiceAccountTokenSecret   = errors.New("obj: secret is not usable for service accounts")
	ErrServiceAccountTokenMissingData = errors.New("obj: service account token secret has no token data")
)

type OpaqueSecret struct {
	Key    client.ObjectKey
	Object *corev1.Secret
}

var _ Loader = &OpaqueSecret{}
var _ Ownable = &OpaqueSecret{}

func (os *OpaqueSecret) Load(ctx context.Context, cl client.Client) (bool, error) {
	ok, err := GetIgnoreNotFound(ctx, cl, os.Key, os.Object)
	if err != nil {
		return false, err
	}

	if os.Object.Type != corev1.SecretTypeOpaque {
		return false, ErrNotOpaqueSecret
	}

	return ok, nil
}

func (os *OpaqueSecret) Owned(ctx context.Context, owner Owner) error {
	return Own(os.Object, owner)
}

func (os *OpaqueSecret) Data(key string) (string, bool) {
	b, found := os.Object.Data[key]
	if !found {
		return "", false
	}

	return string(b), true
}

func NewOpaqueSecret(key client.ObjectKey) *OpaqueSecret {
	return &OpaqueSecret{
		Key: key,
		Object: &corev1.Secret{
			Type: corev1.SecretTypeOpaque,
		},
	}
}

type ImagePullSecret struct {
	Key    client.ObjectKey
	Object *corev1.Secret
}

var _ Persister = &ImagePullSecret{}
var _ Loader = &ImagePullSecret{}
var _ Ownable = &ImagePullSecret{}

func (ips *ImagePullSecret) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, ips.Key, ips.Object)
}

func (ips *ImagePullSecret) Load(ctx context.Context, cl client.Client) (bool, error) {
	ok, err := GetIgnoreNotFound(ctx, cl, ips.Key, ips.Object)
	if err != nil {
		return false, err
	}

	if ips.Object.Type != corev1.SecretTypeDockerConfigJson {
		return false, ErrNotImagePullSecret
	}

	return ok, nil
}

func (ips *ImagePullSecret) Owned(ctx context.Context, owner Owner) error {
	return Own(ips.Object, owner)
}

func NewImagePullSecret(key client.ObjectKey) *ImagePullSecret {
	return &ImagePullSecret{
		Key: key,
		Object: &corev1.Secret{
			Type: corev1.SecretTypeDockerConfigJson,
		},
	}
}

func ConfigureImagePullSecret(target, src *ImagePullSecret) {
	target.Object.Data = src.Object.DeepCopy().Data
}

type ServiceAccountTokenSecret struct {
	Key    client.ObjectKey
	Object *corev1.Secret
}

var _ Persister = &ImagePullSecret{}
var _ Loader = &ImagePullSecret{}

func (sats *ServiceAccountTokenSecret) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, sats.Key, sats.Object)
}

func (sats *ServiceAccountTokenSecret) Load(ctx context.Context, cl client.Client) (bool, error) {
	ok, err := GetIgnoreNotFound(ctx, cl, sats.Key, sats.Object)
	if err != nil {
		return false, err
	}

	if sats.Object.Type != corev1.SecretTypeServiceAccountToken {
		return false, ErrNotServiceAccountTokenSecret
	}

	return ok, nil
}

func (sats *ServiceAccountTokenSecret) Token() (string, error) {
	tok := string(sats.Object.Data["token"])
	if tok == "" {
		return "", ErrServiceAccountTokenMissingData
	}

	return tok, nil
}

func NewServiceAccountTokenSecret(key client.ObjectKey) *ServiceAccountTokenSecret {
	return &ServiceAccountTokenSecret{
		Key: key,
		Object: &corev1.Secret{
			Type: corev1.SecretTypeServiceAccountToken,
		},
	}
}
