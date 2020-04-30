package config

// WorkflowControllerConfig is the configuration object used to
// configure the Workflow controller.
type WorkflowControllerConfig struct {
	Namespace               string
	ImagePullSecret         string
	WhenConditionsImage     string
	MaxConcurrentReconciles int
}

// K8sClusterProvisionerConfig is the configuration object to used
// to configure the Kubernetes provisioner task.
type K8sClusterProvisionerConfig struct {
	WorkDir string
}
