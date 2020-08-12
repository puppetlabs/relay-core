package workflow

import (
	"path"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	"github.com/puppetlabs/relay-core/pkg/model"
	v1 "github.com/puppetlabs/relay-core/pkg/workflow/types/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesObjectMapping is a result struct that contains the kubernets
// objects created from translating a WorkflowData object.
type KubernetesObjectMapping struct {
	Namespace      *corev1.Namespace
	Tenant         *v1beta1.Tenant
	WorkflowRun    *nebulav1.WorkflowRun
	WebhookTrigger *v1beta1.WebhookTrigger
}

// KubernetesEngineMapper translates a v1.WorkflowData object into a kubernets
// object manifest. The results have not been applied or created on the
// kubernetes server.
type KubernetesEngineMapper interface {
	ToRuntimeObjectsManifest(*v1.WorkflowData) (*KubernetesObjectMapping, error)
}

const (
	defaultWorkflowName    = "workflow"
	defaultWorkflowRunName = "workflow-run"
	defaultNamespace       = "default"
)

// DefaultEngineMapperOption is a func that takes a *DefaultEngineMapper and
// configures it. Each function is responsible for a small configuration, such
// as setting the name field.
type DefaultEngineMapperOption func(*DefaultEngineMapper)

func WithDomainID(id string) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.domainID = id
	}
}

func WithWorkflowName(name string) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.name = name
	}
}

func WithWorkflowRunName(name string) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.runName = name
	}
}

func WithNamespace(ns string) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.namespace = ns
	}
}

func WithRunParameters(params v1.WorkflowRunParameters) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.runParameters = params
	}
}

func WithVaultEngineMount(mount string) DefaultEngineMapperOption {
	return func(m *DefaultEngineMapper) {
		m.vaultEngineMount = mount
	}
}

// DefaultEngineMapper maps a v1.WorkflowRun to Kubernetes runtime objects. It
// is the default for relay-operator.
type DefaultEngineMapper struct {
	name             string
	runName          string
	namespace        string
	runParameters    v1.WorkflowRunParameters
	domainID         string
	vaultEngineMount string
}

// ToRuntimeObjectsManifest returns a KubernetesObjectMapping that contains
// uncreated objects that map to relay-core CRDs and other kubernetes resources
// required to support a run.
func (m *DefaultEngineMapper) ToRuntimeObjectsManifest(wd *v1.WorkflowData) (*KubernetesObjectMapping, error) {
	manifest := KubernetesObjectMapping{}

	if m.namespace != defaultNamespace {
		manifest.Namespace = mapNamespace(m.namespace)
	}

	wp := map[string]interface{}{}
	for k, v := range wd.Parameters {
		wp[k] = v.Default
	}

	wrp := map[string]interface{}{}
	for k, v := range m.runParameters {
		wrp[k] = v.Value
	}

	annotations := map[string]string{
		model.RelayDomainIDAnnotation: m.domainID,
		model.RelayTenantIDAnnotation: m.name,
	}

	if m.vaultEngineMount != "" {
		annotations[model.RelayVaultEngineMountAnnotation] = m.vaultEngineMount
		annotations[model.RelayVaultSecretPathAnnotation] = path.Join("workflows", m.name)
		annotations[model.RelayVaultConnectionPathAnnotation] = path.Join("connections", m.domainID)
	}

	manifest.WorkflowRun = &nebulav1.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.runName,
			Namespace: m.namespace,
			// TODO
			Annotations: annotations,
		},
		Spec: nebulav1.WorkflowRunSpec{
			Name:       m.runName,
			Parameters: v1beta1.NewUnstructuredObject(wrp),
			Workflow: nebulav1.Workflow{
				Name:       m.name,
				Parameters: v1beta1.NewUnstructuredObject(wp),
				Steps:      mapSteps(wd),
			},
		},
	}

	return &manifest, nil
}

// NewDefaultEngineMapper takes any number of DefaultEngineMapperOption's and
// returns a configured KubernetesEngineMapper.
func NewDefaultEngineMapper(opts ...DefaultEngineMapperOption) *DefaultEngineMapper {
	m := &DefaultEngineMapper{
		name:      defaultWorkflowName,
		runName:   defaultWorkflowRunName,
		namespace: defaultNamespace,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func mapNamespace(ns string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "nebula",
				// This label controls how network policies external to the API
				// allow access to the Vault server. Do not remove it!
				"nebula.puppet.com/network-policy.customer": "true",
				// This label controls RBAC access to the namespace by the
				// controller.
				"controller.relay.sh/tenant-workload": "true",
			},
		},
	}
}

func mapSteps(wd *v1.WorkflowData) []*nebulav1.WorkflowStep {
	var workflowSteps []*nebulav1.WorkflowStep

	for _, value := range wd.Steps {
		workflowStep := nebulav1.WorkflowStep{
			Name:      value.Name,
			DependsOn: value.DependsOn,
			When:      v1beta1.AsUnstructured(value.When.Tree),
		}

		switch variant := value.Variant.(type) {
		case *v1.ContainerWorkflowStep:
			workflowStep.Image = variant.Image
			workflowStep.Spec = mapStepSpec(variant.Spec)
			workflowStep.Input = variant.Input
			workflowStep.Command = variant.Command
			workflowStep.Args = variant.Args
		}

		workflowSteps = append(workflowSteps, &workflowStep)
	}

	return workflowSteps
}

func mapStepSpec(jm map[string]serialize.JSONTree) v1beta1.UnstructuredObject {
	uo := make(v1beta1.UnstructuredObject, len(jm))
	for k, v := range jm {
		// The inner data type has to be compatible with transfer.JSONInterface
		// here, hence the explicit cast to interface{}.
		uo[k] = v1beta1.AsUnstructured(interface{}(v.Tree))
	}

	return uo
}
