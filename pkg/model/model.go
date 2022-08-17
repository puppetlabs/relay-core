package model

import (
	"time"
)

const (
	DefaultImage   = "alpine:latest"
	DefaultCommand = "echo"

	InputScriptMountPath = "/var/run/puppet/relay/config"
	InputScriptName      = "input-script"

	// TODO Consider configuration options for runtime tools
	ToolsCommandInitialize = "initialize"
	ToolsImage             = "us-docker.pkg.dev/puppet-relay-contrib-oss/relay-core/relay-runtime-tools:latest"
	ToolsMountName         = "relay-tools"
	ToolsMountPath         = "/var/lib/puppet/relay"
	ToolsSource            = "/ko-app/relay-runtime-tools"
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

	RelayInstallerNameLabel = "install.relay.sh/relay-core"
	RelayAppNameLabel       = "app.kubernetes.io/name"
	RelayAppInstanceLabel   = "app.kubernetes.io/instance"
	RelayAppComponentLabel  = "app.kubernetes.io/component"
	RelayAppManagedByLabel  = "app.kubernetes.io/managed-by"
)

type EnvironmentVariable string

const (
	EnvironmentVariableDefaultTimeout      EnvironmentVariable = "RELAY_DEFAULT_TIMEOUT"
	EnvironmentVariableEnableSecureLogging EnvironmentVariable = "RELAY_ENABLE_SECURE_LOGGING"
	EnvironmentVariableMetadataAPIURL      EnvironmentVariable = "METADATA_API_URL"
)

func (ev EnvironmentVariable) String() string {
	return string(ev)
}

type DeploymentEnvironment struct {
	name          string
	secureLogging bool
	timeout       time.Duration
}

func (e DeploymentEnvironment) Name() string {
	return e.name
}

func (e DeploymentEnvironment) SecureLogging() bool {
	return e.secureLogging
}

func (e DeploymentEnvironment) Timeout() time.Duration {
	return e.timeout
}

var (
	DeploymentEnvironmentDevelopment = DeploymentEnvironment{
		name:          "dev",
		secureLogging: true,
		timeout:       1 * time.Minute,
	}
	DeploymentEnvironmentTest = DeploymentEnvironment{
		name:          "test",
		secureLogging: false,
		timeout:       5 * time.Second,
	}

	DeploymentEnvironments = map[string]DeploymentEnvironment{
		DeploymentEnvironmentDevelopment.Name(): DeploymentEnvironmentDevelopment,
		DeploymentEnvironmentTest.Name():        DeploymentEnvironmentTest,
	}
)

// MetadataManagers are the managers used by actions accessing the metadata
// service.
type MetadataManagers interface {
	ActionStatus() ActionStatusManager
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
	StepDecorators() StepDecoratorManager
	StepMessages() StepMessageManager
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
}

// WebhookTriggerReconcilerManagers are the managers used by the workflow
// trigger reconciler when configuring a Knative service for the trigger.
type WebhookTriggerReconcilerManagers interface {
	Environment() EnvironmentSetterManager
	Spec() SpecSetterManager
}
