package builder

import (
	"github.com/puppetlabs/relay-core/pkg/manager/reject"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type metadataManagers struct {
	actionMetadata model.ActionMetadataManager
	actionStatus   model.ActionStatusManager
	connections    model.ConnectionManager
	conditions     model.ConditionGetterManager
	events         model.EventManager
	environment    model.EnvironmentGetterManager
	logs           model.LogManager
	parameters     model.ParameterGetterManager
	secrets        model.SecretManager
	spec           model.SpecGetterManager
	state          model.StateGetterManager
	stepDecorators model.StepDecoratorManager
	stepMessages   model.StepMessageManager
	stepOutputs    model.StepOutputManager
	workflowRuns   model.WorkflowRunManager
}

var _ model.MetadataManagers = &metadataManagers{}

func (mm *metadataManagers) ActionMetadata() model.ActionMetadataManager {
	return mm.actionMetadata
}

func (mm *metadataManagers) ActionStatus() model.ActionStatusManager {
	return mm.actionStatus
}

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

func (mm *metadataManagers) Logs() model.LogManager {
	return mm.logs
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

func (mm *metadataManagers) StepDecorators() model.StepDecoratorManager {
	return mm.stepDecorators
}

func (mm *metadataManagers) StepMessages() model.StepMessageManager {
	return mm.stepMessages
}

func (mm *metadataManagers) StepOutputs() model.StepOutputManager {
	return mm.stepOutputs
}

func (mm *metadataManagers) WorkflowRuns() model.WorkflowRunManager {
	return mm.workflowRuns
}

type MetadataBuilder struct {
	actionMetadata model.ActionMetadataManager
	actionStatus   model.ActionStatusManager
	connections    model.ConnectionManager
	conditions     model.ConditionGetterManager
	events         model.EventManager
	environment    model.EnvironmentGetterManager
	logs           model.LogManager
	parameters     model.ParameterGetterManager
	secrets        model.SecretManager
	spec           model.SpecGetterManager
	state          model.StateGetterManager
	stepDecorators model.StepDecoratorManager
	stepMessages   model.StepMessageManager
	stepOutputs    model.StepOutputManager
	workflowRuns   model.WorkflowRunManager
}

func (mb *MetadataBuilder) SetActionMetadata(m model.ActionMetadataManager) *MetadataBuilder {
	mb.actionMetadata = m
	return mb
}

func (mb *MetadataBuilder) SetActionStatus(m model.ActionStatusManager) *MetadataBuilder {
	mb.actionStatus = m
	return mb
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

func (mb *MetadataBuilder) SetLogs(m model.LogManager) *MetadataBuilder {
	mb.logs = m
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

func (mb *MetadataBuilder) SetStepDecorators(m model.StepDecoratorManager) *MetadataBuilder {
	mb.stepDecorators = m
	return mb
}

func (mb *MetadataBuilder) SetStepMessages(m model.StepMessageManager) *MetadataBuilder {
	mb.stepMessages = m
	return mb
}

func (mb *MetadataBuilder) SetStepOutputs(m model.StepOutputManager) *MetadataBuilder {
	mb.stepOutputs = m
	return mb
}

func (mb *MetadataBuilder) SetWorkflowRuns(m model.WorkflowRunManager) *MetadataBuilder {
	mb.workflowRuns = m
	return mb
}

func (mb *MetadataBuilder) Build() model.MetadataManagers {
	return &metadataManagers{
		actionMetadata: mb.actionMetadata,
		actionStatus:   mb.actionStatus,
		connections:    mb.connections,
		conditions:     mb.conditions,
		events:         mb.events,
		environment:    mb.environment,
		logs:           mb.logs,
		parameters:     mb.parameters,
		secrets:        mb.secrets,
		spec:           mb.spec,
		state:          mb.state,
		stepDecorators: mb.stepDecorators,
		stepMessages:   mb.stepMessages,
		stepOutputs:    mb.stepOutputs,
		workflowRuns:   mb.workflowRuns,
	}
}

func NewMetadataBuilder() *MetadataBuilder {
	return &MetadataBuilder{
		actionStatus:   reject.ActionStatusManager,
		actionMetadata: reject.ActionMetadataManager,
		connections:    reject.ConnectionManager,
		conditions:     reject.ConditionManager,
		events:         reject.EventManager,
		environment:    reject.EnvironmentManager,
		logs:           reject.LogManager,
		parameters:     reject.ParameterManager,
		secrets:        reject.SecretManager,
		spec:           reject.SpecManager,
		state:          reject.StateManager,
		stepDecorators: reject.StepDecoratorManager,
		stepMessages:   reject.StepMessageManager,
		stepOutputs:    reject.StepOutputManager,
		workflowRuns:   reject.WorkflowRunManager,
	}
}
