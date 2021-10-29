package model

const (
	DefaultImage = "alpine:latest"

	// TODO All tool injection settings should be fully configurable

	ToolsMountPath = "/var/lib/puppet/relay"
)

const (
	RelayDomainIDAnnotation            = "relay.sh/domain-id"
	RelayTenantIDAnnotation            = "relay.sh/tenant-id"
	RelayVaultEngineMountAnnotation    = "relay.sh/vault-engine-mount"
	RelayVaultSecretPathAnnotation     = "relay.sh/vault-secret-path"
	RelayVaultConnectionPathAnnotation = "relay.sh/vault-connection-path"

	RelayControllerTokenHashAnnotation = "controller.relay.sh/token-hash"

	RelayControllerTenantNameLabel       = "controller.relay.sh/tenant-name"
	RelayControllerTenantWorkloadLabel   = "controller.relay.sh/tenant-workload"
	RelayControllerWorkflowRunIDLabel    = "controller.relay.sh/run-id"
	RelayControllerWebhookTriggerIDLabel = "controller.relay.sh/webhook-trigger-id"
)

// MetadataManagers are the managers used by actions accessing the metadata
// service.
type MetadataManagers interface {
	Conditions() ConditionGetterManager
	Connections() ConnectionManager
	Events() EventManager
	Environment() EnvironmentGetterManager
	Parameters() ParameterGetterManager
	Logs() LogManager
	Secrets() SecretManager
	Spec() SpecGetterManager
	State() StateGetterManager
	Timers() TimerSetterManager
	ActionMetadata() ActionMetadataManager
	StepDecorators() StepDecoratorManager
	StepOutputs() StepOutputManager
	WorkflowRuns() WorkflowRunManager
}

// RunReconcilerManagers are the managers used by the run reconciler when
// setting up a new run.
type RunReconcilerManagers interface {
	Conditions() ConditionSetterManager
	Parameters() ParameterSetterManager
	Environment() EnvironmentSetterManager
	Spec() SpecSetterManager
	State() StateSetterManager
	Timers() TimerGetterManager
}

// WebhookTriggerReconcilerManagers are the managers used by the workflow
// trigger reconciler when configuring a Knative service for the trigger.
type WebhookTriggerReconcilerManagers interface {
	Environment() EnvironmentSetterManager
	Spec() SpecSetterManager
}
