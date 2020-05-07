package model

// MetadataManagers are the managers used by actions accessing the metadata
// service.
type MetadataManagers interface {
	Conditions() ConditionGetterManager
	Secrets() SecretManager
	Spec() SpecGetterManager
	State() StateGetterManager
	StepOutputs() StepOutputManager
}

// RunReconcilerManagers are the managers used by the run reconciler when
// setting up a new run.
type RunReconcilerManagers interface {
	Conditions() ConditionSetterManager
	Spec() SpecSetterManager
	State() StateSetterManager
}

// WebhookTriggerReconcilerManagers are the managers used by the workflow
// trigger reconciler when configuring a Knative service for the trigger.
type WebhookTriggerReconcilerManagers interface {
	Spec() SpecSetterManager
}