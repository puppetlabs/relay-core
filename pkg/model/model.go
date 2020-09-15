package model

const (
	DefaultImage              = "alpine:latest"
	DefaultToolInjectionImage = "relaysh/relay-runtime-tools"

	// TODO All tool injection settings should be fully configurable
	ToolInjectionImagePath = "/relay/runtime/tools/."
	ToolInjectionMountName = "relay-runtime-tools"
	ToolInjectionMountPath = "/var/lib/puppet/relay/"

	ToolInjectionVolumeClaimSuffixReadOnlyMany  = "-volume-rox"
	ToolInjectionVolumeClaimSuffixReadWriteOnce = "-volume-rwo"
)

const (
	RelayDomainIDAnnotation            = "relay.sh/domain-id"
	RelayTenantIDAnnotation            = "relay.sh/tenant-id"
	RelayVaultEngineMountAnnotation    = "relay.sh/vault-engine-mount"
	RelayVaultSecretPathAnnotation     = "relay.sh/vault-secret-path"
	RelayVaultConnectionPathAnnotation = "relay.sh/vault-connection-path"

	RelayControllerTokenHashAnnotation        = "controller.relay.sh/token-hash"
	RelayControllerDependencyOfAnnotation     = "controller.relay.sh/dependency-of"
	RelayControllerToolsVolumeClaimAnnotation = "controller.relay.sh/tools-volume-claim"

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
	Secrets() SecretManager
	Spec() SpecGetterManager
	State() StateGetterManager
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
