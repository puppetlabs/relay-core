package app

import (
	"context"
	"fmt"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	VaultTokenVarName     = "VAULT_TOKEN"
	VaultUnsealKeyVarName = "VAULT_UNSEAL_KEY"
)

type VaultConfigAuth struct {
	Auth        *v1alpha1.VaultConfigAuth
	TokenSecret *corev1obj.Secret
}

var _ lifecycle.Loader = &VaultConfigAuth{}

func (v *VaultConfigAuth) Load(ctx context.Context, cl client.Client) (bool, error) {
	klog.Infof("loading vault config auth secret %s", v.TokenSecret.Key.String())

	ok, err := lifecycle.IgnoreNilLoader{v.TokenSecret}.Load(ctx, cl)
	if err != nil {
		return ok, fmt.Errorf("failed to load vault config auth: %w", err)
	}

	return ok, nil
}

func (v *VaultConfigAuth) TokenEnvVar() (corev1.EnvVar, bool) {
	if v.Auth.Token != "" {
		return corev1.EnvVar{
			Name:  VaultTokenVarName,
			Value: v.Auth.Token,
		}, true
	} else if v.TokenSecret != nil {
		return corev1.EnvVar{
			Name: VaultTokenVarName,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: v.Auth.TokenFrom.SecretKeyRef,
			},
		}, true
	}

	return corev1.EnvVar{}, false
}

func (v *VaultConfigAuth) UnsealKeyEnvVar() (corev1.EnvVar, bool) {
	if v.Auth.UnsealKey != "" {
		return corev1.EnvVar{
			Name:  VaultUnsealKeyVarName,
			Value: v.Auth.UnsealKey,
		}, true
	}

	return corev1.EnvVar{}, false
}

func NewVaultConfigAuth(c *obj.Core, auth *v1alpha1.VaultConfigAuth) *VaultConfigAuth {
	v := &VaultConfigAuth{
		Auth: auth,
	}

	if auth.TokenFrom != nil && auth.TokenFrom.SecretKeyRef != nil {
		v.TokenSecret = corev1obj.NewSecret(client.ObjectKey{
			Namespace: c.Object.GetNamespace(),
			Name:      auth.TokenFrom.SecretKeyRef.Name,
		})
	}

	return v
}
