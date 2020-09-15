package v1

import (
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DefaultWebhookTriggerEngineMapperOption func(*DefaultWebhookTriggerEngineMapper)

func WithIDWebhookTriggerOption(id string) DefaultWebhookTriggerEngineMapperOption {
	return func(m *DefaultWebhookTriggerEngineMapper) {
		m.id = id
	}
}

func WithNameWebhookTriggerOption(name string) DefaultWebhookTriggerEngineMapperOption {
	return func(m *DefaultWebhookTriggerEngineMapper) {
		m.name = name
	}
}

func WithImageWebhookTriggerOption(image string) DefaultWebhookTriggerEngineMapperOption {
	return func(m *DefaultWebhookTriggerEngineMapper) {
		m.image = image
	}
}

type DefaultWebhookTriggerEngineMapper struct {
	id    string
	name  string
	image string
}

func (m *DefaultWebhookTriggerEngineMapper) ToRuntimeObjectsManifest(tenant *v1beta1.Tenant, source *WebhookWorkflowTriggerSource) (*WebhookTriggerKubernetesObjectMapping, error) {
	wt := &v1beta1.WebhookTrigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("trigger-%s", m.id),
			Namespace: tenant.GetNamespace(),
			Labels: map[string]string{
				WorkflowTriggerIDLabel:   m.id,
				WorkflowTriggerNameLabel: m.name,
			},
			Annotations: map[string]string{
				// Note this is the version *we* applied, not necessarily the
				// most current version.
				"managed.relay.sh/tenant.resource-version": tenant.GetResourceVersion(),
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tenant, v1beta1.TenantKind),
			},
		},
		Spec: v1beta1.WebhookTriggerSpec{
			TenantRef: corev1.LocalObjectReference{
				Name: tenant.GetName(),
			},
			Name:    m.name,
			Image:   m.image,
			Input:   source.Input,
			Command: source.Command,
			Args:    source.Args,
			Spec:    mapStepSpec(source.Spec),
		},
	}

	return &WebhookTriggerKubernetesObjectMapping{
		WebhookTrigger: wt,
	}, nil
}

func NewDefaultWebhookTriggerEngineMapper(opts ...DefaultWebhookTriggerEngineMapperOption) *DefaultWebhookTriggerEngineMapper {
	m := &DefaultWebhookTriggerEngineMapper{}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
