package model

const (
	DefaultImage              = "alpine:latest"
	DefaultToolInjectionImage = "relaysh/relay-runtime-tools"

	// TODO All tool injection settings should be fully configurable
	ToolInjectionImagePath = "/relay/runtime/tools/."
	ToolInjectionMountName = "relay-runtime-tools"
	ToolInjectionMountPath = "/var/lib/puppet/relay/"

	ToolInjectionVolumeClaimSuffixReadOnlyMany  = "-inject"
	ToolInjectionVolumeClaimSuffixReadWriteOnce = "-init"
)

const (
	RelayAccountIDLabel           = "managed.relay.sh/account-id"
	RelayWorkflowIDLabel          = "managed.relay.sh/workflow-id"
	RelayWorkflowNameLabel        = "managed.relay.sh/workflow-name"
	RelayWorkflowTriggerIDLabel   = "managed.relay.sh/workflow-trigger-id"
	RelayWorkflowTriggerNameLabel = "managed.relay.sh/workflow-trigger-name"

	RelayExpirationTimeAnnotation = "managed.relay.sh/expiration-time"

	RelayDomainIDAnnotation            = "relay.sh/domain-id"
	RelayTenantIDAnnotation            = "relay.sh/tenant-id"
	RelayVaultEngineMountAnnotation    = "relay.sh/vault-engine-mount"
	RelayVaultSecretPathAnnotation     = "relay.sh/vault-secret-path"
	RelayVaultConnectionPathAnnotation = "relay.sh/vault-connection-path"

	RelayControllerTokenHashAnnotation                = "controller.relay.sh/token-hash"
	RelayControllerDependencyOfAnnotation             = "controller.relay.sh/dependency-of"
	RelayControllerToolInjectionImageDigestAnnotation = "controller.relay.sh/tool-injection-image-digest"
	RelayControllerToolsVolumeAnnotation              = "controller.relay.sh/tools-volume"
	RelayControllerToolsVolumeClaimAnnotation         = "controller.relay.sh/tools-volume-claim"

	RelayControllerTenantNameLabel          = "controller.relay.sh/tenant-name"
	RelayControllerTenantWorkloadLabel      = "controller.relay.sh/tenant-workload"
	RelayControllerToolInjectionVolumeLabel = "controller.relay.sh/tool-injection"
	RelayControllerWorkflowRunIDLabel       = "controller.relay.sh/run-id"
	RelayControllerWebhookTriggerIDLabel    = "controller.relay.sh/webhook-trigger-id"
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
	ActionMetadata() ActionMetadataManager
	StepOutputs() StepOutputManager
}

// RunReconcilerManagers are the managers used by the run reconciler when
// setting up a new run.
type RunReconcilerManagers interface {
	Conditions() ConditionSetterManager
	Parameters() ParameterSetterManager
	Environment() EnvironmentSetterManager
	Spec() SpecSetterManager
	State() StateSetterManager
}

// WebhookTriggerReconcilerManagers are the managers used by the workflow
// trigger reconciler when configuring a Knative service for the trigger.
type WebhookTriggerReconcilerManagers interface {
	Environment() EnvironmentSetterManager
	Spec() SpecSetterManager
}
