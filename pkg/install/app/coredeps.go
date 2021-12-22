package app

import (
	"context"
	"fmt"
	"net/url"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/install/jwt"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultPrivateJWTSigningKeyName = "private-key.pem"
	defaultPublicJWTSigningKeyName  = "public-key.pem"
)

type CoreDepsLoadResult struct {
	VaultConfig *VaultConfigDepsLoadResult
	All         bool
}

type CoreDeps struct {
	Core                *obj.Core
	OwnerConfigMap      *corev1obj.ConfigMap
	Namespace           *corev1obj.Namespace
	OperatorDeps        *OperatorDeps
	MetadataAPIDeps     *MetadataAPIDeps
	LogServiceDeps      *LogServiceDeps
	JWTSigningKeySecret *corev1obj.Secret
	VaultConfigDeps     *VaultConfigDeps
}

func (cd *CoreDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if cd.OwnerConfigMap == nil || cd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(
		cd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: cd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {
		return false, err
	} else if ok {
		return cd.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (cd *CoreDeps) Load(ctx context.Context, cl client.Client) (*CoreDepsLoadResult, error) {
	if _, err := cd.Core.Load(ctx, cl); err != nil {
		return nil, err
	}

	cd.VaultConfigDeps = NewVaultConfigDeps(cd.Core)

	vr, err := cd.VaultConfigDeps.Load(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault config deps: %w", err)
	}

	cd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(cd.Core.Key, "owner"))
	cd.OperatorDeps = NewOperatorDeps(cd.Core)
	cd.MetadataAPIDeps = NewMetadataAPIDeps(cd.Core)
	cd.LogServiceDeps = NewLogServiceDeps(cd.Core)
	cd.JWTSigningKeySecret = corev1obj.NewSecret(client.ObjectKey{
		Name:      cd.Core.Object.Spec.JWTSigningKeyRef.Name,
		Namespace: cd.Core.Key.Namespace,
	})

	ok, err := lifecycle.Loaders{
		lifecycle.RequiredLoader{Loader: cd.Namespace},
		cd.OwnerConfigMap,
		cd.OperatorDeps,
		cd.MetadataAPIDeps,
		cd.LogServiceDeps,
		cd.JWTSigningKeySecret,
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &CoreDepsLoadResult{All: ok, VaultConfig: vr}, nil
}

func (cd *CoreDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := cd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		cd.OperatorDeps.OwnerConfigMap,
		cd.MetadataAPIDeps.OwnerConfigMap,
		cd.LogServiceDeps.OwnerConfigMap,
		cd.VaultConfigDeps.OwnerConfigMap,
	}

	ps := []lifecycle.Persister{
		cd.Namespace,
		cd.MetadataAPIDeps,
		cd.OperatorDeps,
		cd.LogServiceDeps,
		cd.VaultConfigDeps,
	}

	// if we didn't load a secret managed outside the installer, we are
	// generating keys here and therefore want to fully manage the secret
	// object by owning and persisting it.
	if cd.JWTSigningKeySecret.Object.GetUID() == "" {
		os = append(os, cd.JWTSigningKeySecret)
		ps = append(ps, cd.JWTSigningKeySecret)
	}

	for _, o := range os {
		if err := cd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewCoreDeps(c *obj.Core) *CoreDeps {
	return &CoreDeps{
		Core:      c,
		Namespace: corev1obj.NewNamespace(c.Object.GetNamespace()),
	}
}

func ApplyCoreDeps(ctx context.Context, cl client.Client, c *obj.Core) (*CoreDeps, error) {
	cd := NewCoreDeps(c)

	_, err := cd.Load(ctx, cl)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	if err := ConfigureCoreDeps(cd); err != nil {
		klog.Error(err)
		return nil, err
	}

	ConfigureCoreDefaults(cd)

	if err := cd.Core.Persist(ctx, cl); err != nil {
		klog.Error(err)
		return nil, err
	}

	// if result.VaultConfig.JobsRunning {
	// 	return nil, errors.New("waiting for vault configuration jobs to finish")
	// } else if !result.VaultConfig.JobsExist {
	// 	if err := ConfigureVaultConfigDeps(cd.VaultConfigDeps); err != nil {
	// 		return nil, err
	// 	}

	// 	if err := cd.VaultConfigDeps.Persist(ctx, cl); err != nil {
	// 		return nil, err
	// 	}

	// 	return nil, errors.New("waiting for vault configuration jobs to be created")
	// }

	klog.Info("configuring vault deps")
	if err := ConfigureVaultConfigDeps(cd.VaultConfigDeps); err != nil {
		return nil, err
	}

	klog.Info("configuring operator deps")
	if err := ConfigureOperatorDeps(ctx, cd.OperatorDeps); err != nil {
		return nil, err
	}

	klog.Info("configuring metadata-api deps")
	if err := ConfigureMetadataAPIDeps(ctx, cd.MetadataAPIDeps); err != nil {
		return nil, err
	}

	klog.Info("configuring log-service deps")
	if err := ConfigureLogServiceDeps(ctx, cd.LogServiceDeps); err != nil {
		return nil, err
	}

	klog.Info("persisting all deps")
	if err := cd.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return cd, nil
}

func ConfigureCoreDeps(cd *CoreDeps) error {
	if err := DependencyManager.SetDependencyOf(
		cd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: cd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	if cd.JWTSigningKeySecret.Object.GetUID() == "" {
		if err := ConfigureJWTSigningKeys(cd.JWTSigningKeySecret); err != nil {
			return err
		}
	}

	return nil
}

func ConfigureCoreDefaults(cd *CoreDeps) {
	core := cd.Core

	if core.Object.Spec.Operator == nil {
		core.Object.Spec.Operator = &v1alpha1.OperatorConfig{}
	}

	if core.Object.Spec.Operator.ServiceAccountName == "" {
		core.Object.Spec.Operator.ServiceAccountName = cd.OperatorDeps.ServiceAccount.Key.Name
	}

	if core.Object.Spec.JWTSigningKeyRef == nil {
		resourceKey := helper.SuffixObjectKey(core.Key, "jwt-signing-keys")

		core.Object.Spec.JWTSigningKeyRef = &v1alpha1.JWTSigningKeySource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: resourceKey.Name,
			},
			PrivateKeyRef: defaultPrivateJWTSigningKeyName,
			PublicKeyRef:  defaultPublicJWTSigningKeyName,
		}
	}

	if core.Object.Spec.MetadataAPI == nil {
		core.Object.Spec.MetadataAPI = &v1alpha1.MetadataAPIConfig{}
	}

	if core.Object.Spec.MetadataAPI.ServiceAccountName == "" {
		core.Object.Spec.MetadataAPI.ServiceAccountName = cd.MetadataAPIDeps.ServiceAccount.Key.Name
	}

	if core.Object.Spec.MetadataAPI.URL == nil {
		u := url.URL{
			Scheme: "http",
		}

		if core.Object.Spec.MetadataAPI.TLSSecretName != nil {
			u.Scheme = "https"
		}

		u.Host = fmt.Sprintf("%s.%s.svc.cluster.local", cd.MetadataAPIDeps.Service.Key.Name, cd.MetadataAPIDeps.Service.Key.Namespace)

		us := u.String()
		core.Object.Spec.MetadataAPI.URL = &us
	}

	if core.Object.Spec.LogService.ServiceAccountName == "" {
		core.Object.Spec.LogService.ServiceAccountName = cd.LogServiceDeps.ServiceAccount.Key.Name
	}
}

func ConfigureJWTSigningKeys(sec *corev1obj.Secret) error {
	pair, err := jwt.GenerateSigningKeys()
	if err != nil {
		return err
	}

	if sec.Object.Data == nil {
		sec.Object.Data = make(map[string][]byte)
	}

	sec.Object.Data[defaultPrivateJWTSigningKeyName] = pair.PrivateKey
	sec.Object.Data[defaultPublicJWTSigningKeyName] = pair.PublicKey

	return nil
}
