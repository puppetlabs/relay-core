package models

// Spec is the nebula workflow spec for provisioning a k8s cluster
type K8sProvisionerSpec struct {
	// Provider is which cloud provider to use (currently gcp or aws)
	Provider string `json:"provider"`
	// Project is which project within the provider account to use
	Project string `json:"project"`
	// Credentials provides api access credentials to the cloud platform sdks
	Credentials Credentials `json:"credentials"`
	// ClusterName is the name of the cluster (including DNS zone)
	ClusterName string `json:"clusterName"`
	// StateStoreName is the name of the storage bucket to create for kops state
	StateStoreName string `json:"stateStoreName"`
	// MasterCount is how many nodes to provision as masters. Setting this value
	// higher than 1 enables HA.
	MasterCount int `json:"masterCount"`
	// NodeCount is how many nodes to provision
	// TODO: autoscaling needs to be figured out
	NodeCount int `json:"nodeCount"`
	// Zones is which zones in the cloud provider to run instances in
	Zones  []string `json:"zones"`
	Region string   `json:"region"`
}

type Credentials struct {
	// GCPServiceAccountFile is a string that represents the file content
	// for cloud auth credentials.
	GCPServiceAccountFile string `json:"gcpServiceAccountFile"`
	// AWSAccessKeyID is the Access Key ID for an AWS account
	AWSAccessKeyID string `json:"awsAccessKeyID"`
	// AWSSecretAccessKey is the Secret Access Key for an AWS account
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
	// SSHPublicKey is a public key to import for use in AWS. This MUST be
	// RSA as that's the only algorithm AWS supports at the moment.
	SSHPublicKey string `json:"sshPublicKey"`
}
