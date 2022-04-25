package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/install/jwt"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultPrivateJWTSigningKeyName = "private-key.pem"
	defaultPublicJWTSigningKeyName  = "public-key.pem"
)

type JWTSigningKeyDeps struct {
	Core                       *obj.Core
	OwnerConfigMap             *corev1obj.ConfigMap
	ManagedJWTSigningKeySecret *corev1obj.Secret
	Labels                     map[string]string
}

func (d *JWTSigningKeyDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if d.OwnerConfigMap == nil || d.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(
		d.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: d.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {
		return false, err
	} else if ok {
		return d.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (d *JWTSigningKeyDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	key := helper.SuffixObjectKey(d.Core.Key, "jwt-signing-keys")

	d.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	if d.Core.Object.Spec.Vault.JWTSigningKeys == nil {
		d.ManagedJWTSigningKeySecret = corev1obj.NewSecret(key)
	}

	ok, err := lifecycle.Loaders{
		d.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: d.ManagedJWTSigningKeySecret},
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *JWTSigningKeyDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(d.OwnerConfigMap.Object, owner)
}

func (d *JWTSigningKeyDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := d.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		lifecycle.IgnoreNilOwnablePersister{OwnablePersister: d.ManagedJWTSigningKeySecret},
	}

	for _, obj := range objs {
		if err := d.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (d *JWTSigningKeyDeps) Configure(_ context.Context) error {
	if err := DependencyManager.SetDependencyOf(
		d.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: d.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	if d.ManagedJWTSigningKeySecret != nil {
		if err := ConfigureJWTSigningKeys(d.ManagedJWTSigningKeySecret); err != nil {
			return err
		}
	}

	return nil
}

func (d *JWTSigningKeyDeps) PrivateKey() corev1.SecretKeySelector {
	if d.ManagedJWTSigningKeySecret != nil {
		return corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: d.ManagedJWTSigningKeySecret.Key.Name,
			},
			Key: defaultPrivateJWTSigningKeyName,
		}
	}

	return d.Core.Object.Spec.Vault.JWTSigningKeys.PrivateKeyRef
}

func (d *JWTSigningKeyDeps) PublicKey() corev1.SecretKeySelector {
	if d.ManagedJWTSigningKeySecret != nil {
		return corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: d.ManagedJWTSigningKeySecret.Key.Name,
			},
			Key: defaultPublicJWTSigningKeyName,
		}
	}

	return d.Core.Object.Spec.Vault.JWTSigningKeys.PublicKeyRef
}

func NewJWTSigningKeyDeps(c *obj.Core) *JWTSigningKeyDeps {
	return &JWTSigningKeyDeps{
		Core: c,
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "jwt-signing-keys",
			model.RelayAppInstanceLabel:   norm.AnyDNSLabelNameSuffixed("jwt-signing-keys-", c.Key.Name),
			model.RelayAppComponentLabel:  "auth",
			model.RelayAppManagedByLabel:  "relay-installer",
		},
	}
}

func ConfigureJWTSigningKeys(sec *corev1obj.Secret) error {
	if sec.Object.Data == nil {
		sec.Object.Data = make(map[string][]byte)
	}

	_, foundExistingPrivateKey := sec.Object.Data[defaultPrivateJWTSigningKeyName]
	_, foundExistingPublicKey := sec.Object.Data[defaultPublicJWTSigningKeyName]

	if foundExistingPrivateKey && foundExistingPublicKey {
		return nil
	}

	pair, err := jwt.GenerateSigningKeys()
	if err != nil {
		return err
	}

	sec.Object.Data[defaultPrivateJWTSigningKeyName] = pair.PrivateKey
	sec.Object.Data[defaultPublicJWTSigningKeyName] = pair.PublicKey

	return nil
}
