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

// RunKubernetesObjectMapping is a result struct that contains the kubernets
// objects created from translating a WorkflowData object.
type RunKubernetesObjectMapping struct {
	Namespace   *corev1.Namespace
	WorkflowRun *nebulav1.WorkflowRun
}

// RunKubernetesEngineMapper translates a v1.WorkflowData object into a kubernets
// object manifest. The results have not been applied or created on the
// kubernetes server.
type RunKubernetesEngineMapper interface {
	ToRuntimeObjectsManifest(*v1.WorkflowData) (*RunKubernetesObjectMapping, error)
}

const (
	defaultWorkflowName    = "workflow"
	defaultWorkflowRunName = "workflow-run"
	defaultNamespace       = "default"
)

// DefaultRunEngineMapperOption is a func that takes a *DefaultRunEngineMapper and
// configures it. Each function is responsible for a small configuration, such
// as setting the name field.
type DefaultRunEngineMapperOption func(*DefaultRunEngineMapper)

func WithDomainID(id string) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.domainID = id
	}
}

func WithWorkflowName(name string) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.name = name
	}
}

func WithWorkflowRunName(name string) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.runName = name
	}
}

func WithNamespace(ns string) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.namespace = ns
	}
}

func WithRunParameters(params v1.WorkflowRunParameters) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.runParameters = params
	}
}

func WithVaultEngineMount(mount string) DefaultRunEngineMapperOption {
	return func(m *DefaultRunEngineMapper) {
		m.vaultEngineMount = mount
	}
}

// DefaultRunEngineMapper maps a v1.WorkflowRun to Kubernetes runtime objects. It
// is the default for relay-operator.
type DefaultRunEngineMapper struct {
	name             string
	runName          string
	namespace        string
	runParameters    v1.WorkflowRunParameters
	domainID         string
	vaultEngineMount string
}

// ToRuntimeObjectsManifest returns a RunKubernetesObjectMapping that contains
// uncreated objects that map to relay-core CRDs and other kubernetes resources
// required to support a run.
func (m *DefaultRunEngineMapper) ToRuntimeObjectsManifest(wd *v1.WorkflowData) (*RunKubernetesObjectMapping, error) {
	manifest := RunKubernetesObjectMapping{}

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

// NewDefaultRunEngineMapper takes any number of DefaultRunEngineMapperOption's and
// returns a configured KubernetesEngineMapper.
func NewDefaultRunEngineMapper(opts ...DefaultRunEngineMapperOption) *DefaultRunEngineMapper {
	m := &DefaultRunEngineMapper{
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
