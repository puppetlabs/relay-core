package v1

import (
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (
	AccountIDLabel         = "managed.relay.sh/account-id"
	WorkflowIDLabel        = "managed.relay.sh/workflow-id"
	WorkflowTriggerIDLabel = "managed.relay.sh/workflow-trigger-id"
)

const (
	defaultWorkflowName    = "workflow"
	defaultWorkflowRunName = "workflow-run"
	defaultNamespace       = "default"
)

// RunKubernetesObjectMapping is a result struct that contains the kubernets
// objects created from translating a WorkflowData object.
type RunKubernetesObjectMapping struct {
	Namespace   *corev1.Namespace
	WorkflowRun *nebulav1.WorkflowRun
}

type TenantKubernetesObjectMapping struct {
	Namespace *corev1.Namespace
	Tenant    *v1beta1.Tenant
}

type WebhookTriggerKubernetesObjectMapping struct {
	WebhookTrigger *v1beta1.WebhookTrigger
}

// RunKubernetesEngineMapper translates a v1.WorkflowData object into a kubernets
// object manifest. The results have not been applied or created on the
// kubernetes server.
type RunKubernetesEngineMapper interface {
	ToRuntimeObjectsManifest(*WorkflowData) (*RunKubernetesObjectMapping, error)
}

type TenantKubernetesEngineMapper interface {
	ToRuntimeObjectsManifest(id string) (*TenantKubernetesObjectMapping, error)
}

type WebhookTriggerKubernetesEngineMapper interface {
	ToRuntimeObjectsManifest(source *WebhookWorkflowTriggerSource) (*WebhookTriggerKubernetesObjectMapping, error)
}
