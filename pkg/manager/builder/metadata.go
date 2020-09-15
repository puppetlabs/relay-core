package builder

import (
	"github.com/puppetlabs/relay-core/pkg/manager/reject"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type metadataManagers struct {
	connections model.ConnectionManager
	conditions  model.ConditionGetterManager
	events      model.EventManager
	environment model.EnvironmentGetterManager
	parameters  model.ParameterGetterManager
	secrets     model.SecretManager
	spec        model.SpecGetterManager
	state       model.StateGetterManager
	stepOutputs model.StepOutputManager
}

var _ model.MetadataManagers = &metadataManagers{}

func (mm *metadataManagers) Connections() model.ConnectionManager {
	return mm.connections
}

func (mm *metadataManagers) Conditions() model.ConditionGetterManager {
	return mm.conditions
}

func (mm *metadataManagers) Events() model.EventManager {
	return mm.events
}

func (mm *metadataManagers) Environment() model.EnvironmentGetterManager {
	return mm.environment
}

func (mm *metadataManagers) Parameters() model.ParameterGetterManager {
	return mm.parameters
}

func (mm *metadataManagers) Secrets() model.SecretManager {
	return mm.secrets
}

func (mm *metadataManagers) Spec() model.SpecGetterManager {
	return mm.spec
}

func (mm *metadataManagers) State() model.StateGetterManager {
	return mm.state
}

func (mm *metadataManagers) StepOutputs() model.StepOutputManager {
	return mm.stepOutputs
}

type MetadataBuilder struct {
	connections model.ConnectionManager
	conditions  model.ConditionGetterManager
	events      model.EventManager
	environment model.EnvironmentGetterManager
	parameters  model.ParameterGetterManager
	secrets     model.SecretManager
	spec        model.SpecGetterManager
	state       model.StateGetterManager
	stepOutputs model.StepOutputManager
}

func (mb *MetadataBuilder) SetConnections(m model.ConnectionManager) *MetadataBuilder {
	mb.connections = m
	return mb
}

func (mb *MetadataBuilder) SetConditions(m model.ConditionGetterManager) *MetadataBuilder {
	mb.conditions = m
	return mb
}

func (mb *MetadataBuilder) SetEvents(m model.EventManager) *MetadataBuilder {
	mb.events = m
	return mb
}

func (mb *MetadataBuilder) SetEnvironment(m model.EnvironmentGetterManager) *MetadataBuilder {
	mb.environment = m
	return mb
}

func (mb *MetadataBuilder) SetParameters(m model.ParameterGetterManager) *MetadataBuilder {
	mb.parameters = m
	return mb
}

func (mb *MetadataBuilder) SetSecrets(m model.SecretManager) *MetadataBuilder {
	mb.secrets = m
	return mb
}

func (mb *MetadataBuilder) SetSpec(m model.SpecGetterManager) *MetadataBuilder {
	mb.spec = m
	return mb
}

func (mb *MetadataBuilder) SetState(m model.StateGetterManager) *MetadataBuilder {
	mb.state = m
	return mb
}

func (mb *MetadataBuilder) SetStepOutputs(m model.StepOutputManager) *MetadataBuilder {
	mb.stepOutputs = m
	return mb
}

func (mb *MetadataBuilder) Build() model.MetadataManagers {
	return &metadataManagers{
		connections: mb.connections,
		conditions:  mb.conditions,
		events:      mb.events,
		environment: mb.environment,
		parameters:  mb.parameters,
		secrets:     mb.secrets,
		spec:        mb.spec,
		state:       mb.state,
		stepOutputs: mb.stepOutputs,
	}
}

func NewMetadataBuilder() *MetadataBuilder {
	return &MetadataBuilder{
		connections: reject.ConnectionManager,
		conditions:  reject.ConditionManager,
		events:      reject.EventManager,
		environment: reject.EnvironmentManager,
		parameters:  reject.ParameterManager,
		secrets:     reject.SecretManager,
		spec:        reject.SpecManager,
		state:       reject.StateManager,
		stepOutputs: reject.StepOutputManager,
	}
}
