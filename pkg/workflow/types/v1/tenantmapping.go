package v1

import (
	"net/url"

	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DefaultTenantEngineMapperOption func(*DefaultTenantEngineMapper)

func WithNameTenantOption(name string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.name = name
	}
}

func WithIDTenantOption(id string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.id = id
	}
}

func WithWorkflowNameTenantOption(name string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.workflowName = name
	}
}

func WithWorkflowIDTenantOption(id string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.workflowID = id
	}
}

func WithNamespaceTenantOption(ns string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.namespace = ns
	}
}

func WithEventURLTenantOption(u *url.URL) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.eventURL = u
	}
}

func WithTokenSecretNameTenantOption(name string) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.tokenSecretName = name
	}
}

func WithToolInjectionTenantOption(enabled bool) DefaultTenantEngineMapperOption {
	return func(m *DefaultTenantEngineMapper) {
		m.enableToolInjection = enabled
	}
}

type DefaultTenantEngineMapper struct {
	id                  string
	name                string
	namespace           string
	workflowID          string
	workflowName        string
	tokenSecretName     string
	eventURL            *url.URL
	enableToolInjection bool
}

func (m *DefaultTenantEngineMapper) ToRuntimeObjectsManifest() (*TenantKubernetesObjectMapping, error) {
	if m.id == "" {
		return nil, MissingTenantIDError
	}

	if m.workflowID == "" {
		return nil, MissingWorkflowIDError
	}

	name := m.name
	if name == "" {
		name = m.workflowID
	}

	var namespace *corev1.Namespace
	if m.namespace != defaultNamespace {
		namespace = mapNamespace(m.namespace)
	}

	tenant := &v1beta1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
			Labels: map[string]string{
				AccountIDLabel:  m.id,
				WorkflowIDLabel: m.workflowID,
			},
			Annotations: map[string]string{},
		},
		Spec: v1beta1.TenantSpec{
			NamespaceTemplate: v1beta1.NamespaceTemplate{
				Metadata: metav1.ObjectMeta{
					Name: name,
				},
			},
		},
	}

	if m.enableToolInjection {
		tenant.Spec.ToolInjection = v1beta1.ToolInjection{
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadOnlyMany,
					},
				},
			},
		}
	}

	if m.eventURL != nil {
		tenant.Spec.TriggerEventSink = v1beta1.TriggerEventSink{
			API: &v1beta1.APITriggerEventSink{
				URL: m.eventURL.String(),
				TokenFrom: &v1beta1.APITokenSource{
					SecretKeyRef: &v1beta1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.tokenSecretName,
						},
						Key: "token",
					},
				},
			},
		}
	}

	return &TenantKubernetesObjectMapping{
		Namespace: namespace,
		Tenant:    tenant,
	}, nil
}

func NewDefaultTenantEngineMapper(opts ...DefaultTenantEngineMapperOption) *DefaultTenantEngineMapper {
	m := &DefaultTenantEngineMapper{
		workflowName: defaultWorkflowName,
		namespace:    defaultNamespace,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
