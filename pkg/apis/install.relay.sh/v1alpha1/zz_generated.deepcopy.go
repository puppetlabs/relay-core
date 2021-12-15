// +build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdmissionWebhookServerConfig) DeepCopyInto(out *AdmissionWebhookServerConfig) {
	*out = *in
	if in.NamespaceSelector != nil {
		in, out := &in.NamespaceSelector, &out.NamespaceSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdmissionWebhookServerConfig.
func (in *AdmissionWebhookServerConfig) DeepCopy() *AdmissionWebhookServerConfig {
	if in == nil {
		return nil
	}
	out := new(AdmissionWebhookServerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LogServiceConfig) DeepCopyInto(out *LogServiceConfig) {
	*out = *in
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LogServiceConfig.
func (in *LogServiceConfig) DeepCopy() *LogServiceConfig {
	if in == nil {
		return nil
	}
	out := new(LogServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetadataAPIConfig) DeepCopyInto(out *MetadataAPIConfig) {
	*out = *in
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.TLSSecretName != nil {
		in, out := &in.TLSSecretName, &out.TLSSecretName
		*out = new(string)
		**out = **in
	}
	if in.URL != nil {
		in, out := &in.URL, &out.URL
		*out = new(string)
		**out = **in
	}
	if in.LogServiceURL != nil {
		in, out := &in.LogServiceURL, &out.LogServiceURL
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetadataAPIConfig.
func (in *MetadataAPIConfig) DeepCopy() *MetadataAPIConfig {
	if in == nil {
		return nil
	}
	out := new(MetadataAPIConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorConfig) DeepCopyInto(out *OperatorConfig) {
	*out = *in
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.JWTSigningKeySecretName != nil {
		in, out := &in.JWTSigningKeySecretName, &out.JWTSigningKeySecretName
		*out = new(string)
		**out = **in
	}
	if in.TenantSandboxingRuntimeClassName != nil {
		in, out := &in.TenantSandboxingRuntimeClassName, &out.TenantSandboxingRuntimeClassName
		*out = new(string)
		**out = **in
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.StorageAddr != nil {
		in, out := &in.StorageAddr, &out.StorageAddr
		*out = new(string)
		**out = **in
	}
	if in.LogStoragePVCName != nil {
		in, out := &in.LogStoragePVCName, &out.LogStoragePVCName
		*out = new(string)
		**out = **in
	}
	if in.ToolInjection != nil {
		in, out := &in.ToolInjection, &out.ToolInjection
		*out = new(ToolInjectionConfig)
		**out = **in
	}
	if in.AdmissionWebhookServer != nil {
		in, out := &in.AdmissionWebhookServer, &out.AdmissionWebhookServer
		*out = new(AdmissionWebhookServerConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorConfig.
func (in *OperatorConfig) DeepCopy() *OperatorConfig {
	if in == nil {
		return nil
	}
	out := new(OperatorConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelayCore) DeepCopyInto(out *RelayCore) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelayCore.
func (in *RelayCore) DeepCopy() *RelayCore {
	if in == nil {
		return nil
	}
	out := new(RelayCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RelayCore) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelayCoreList) DeepCopyInto(out *RelayCoreList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RelayCore, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelayCoreList.
func (in *RelayCoreList) DeepCopy() *RelayCoreList {
	if in == nil {
		return nil
	}
	out := new(RelayCoreList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RelayCoreList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelayCoreSpec) DeepCopyInto(out *RelayCoreSpec) {
	*out = *in
	in.LogService.DeepCopyInto(&out.LogService)
	if in.Operator != nil {
		in, out := &in.Operator, &out.Operator
		*out = new(OperatorConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.MetadataAPI != nil {
		in, out := &in.MetadataAPI, &out.MetadataAPI
		*out = new(MetadataAPIConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Vault != nil {
		in, out := &in.Vault, &out.Vault
		*out = new(VaultConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.SentryDSNSecretName != nil {
		in, out := &in.SentryDSNSecretName, &out.SentryDSNSecretName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelayCoreSpec.
func (in *RelayCoreSpec) DeepCopy() *RelayCoreSpec {
	if in == nil {
		return nil
	}
	out := new(RelayCoreSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RelayCoreStatus) DeepCopyInto(out *RelayCoreStatus) {
	*out = *in
	out.Vault = in.Vault
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RelayCoreStatus.
func (in *RelayCoreStatus) DeepCopy() *RelayCoreStatus {
	if in == nil {
		return nil
	}
	out := new(RelayCoreStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ToolInjectionConfig) DeepCopyInto(out *ToolInjectionConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ToolInjectionConfig.
func (in *ToolInjectionConfig) DeepCopy() *ToolInjectionConfig {
	if in == nil {
		return nil
	}
	out := new(ToolInjectionConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultConfig) DeepCopyInto(out *VaultConfig) {
	*out = *in
	if in.Auth != nil {
		in, out := &in.Auth, &out.Auth
		*out = new(VaultConfigAuth)
		(*in).DeepCopyInto(*out)
	}
	if in.ConfigMapRef != nil {
		in, out := &in.ConfigMapRef, &out.ConfigMapRef
		*out = new(VaultConfigMapSource)
		**out = **in
	}
	if in.Sidecar != nil {
		in, out := &in.Sidecar, &out.Sidecar
		*out = new(VaultSidecar)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultConfig.
func (in *VaultConfig) DeepCopy() *VaultConfig {
	if in == nil {
		return nil
	}
	out := new(VaultConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultConfigAuth) DeepCopyInto(out *VaultConfigAuth) {
	*out = *in
	if in.TokenFrom != nil {
		in, out := &in.TokenFrom, &out.TokenFrom
		*out = new(VaultTokenSource)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultConfigAuth.
func (in *VaultConfigAuth) DeepCopy() *VaultConfigAuth {
	if in == nil {
		return nil
	}
	out := new(VaultConfigAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultConfigMapSource) DeepCopyInto(out *VaultConfigMapSource) {
	*out = *in
	out.LocalObjectReference = in.LocalObjectReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultConfigMapSource.
func (in *VaultConfigMapSource) DeepCopy() *VaultConfigMapSource {
	if in == nil {
		return nil
	}
	out := new(VaultConfigMapSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultSidecar) DeepCopyInto(out *VaultSidecar) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultSidecar.
func (in *VaultSidecar) DeepCopy() *VaultSidecar {
	if in == nil {
		return nil
	}
	out := new(VaultSidecar)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultStatusSummary) DeepCopyInto(out *VaultStatusSummary) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultStatusSummary.
func (in *VaultStatusSummary) DeepCopy() *VaultStatusSummary {
	if in == nil {
		return nil
	}
	out := new(VaultStatusSummary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VaultTokenSource) DeepCopyInto(out *VaultTokenSource) {
	*out = *in
	if in.SecretKeyRef != nil {
		in, out := &in.SecretKeyRef, &out.SecretKeyRef
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VaultTokenSource.
func (in *VaultTokenSource) DeepCopy() *VaultTokenSource {
	if in == nil {
		return nil
	}
	out := new(VaultTokenSource)
	in.DeepCopyInto(out)
	return out
}
