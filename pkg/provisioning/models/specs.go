package models

// Spec is the nebula workflow spec for provisioning a k8s cluster
type K8sProvisionerSpec struct {
	// Provider is which cloud provider to use (currently gcp or aws)
	Provider string `json:"provider"`
	// Project is which project within the provider account to use
	Project string `json:"project"`
	// Credentials is a base64 string that represents the file content
	// for cloud auth credentials
	Credentials string `json:"credentials"`
	// ClusterName is the name of the cluster (including DNS zone)
	ClusterName string `json:"clusterName"`
	// StateStoreName is the name of the storage bucket to create for kops state
	StateStoreName string `json:"stateStoreName"`
	// NodeCount is how many nodes to provision
	// TODO: autoscaling needs to be figured out
	NodeCount int `json:"nodeCount"`
	// Zones is which zones in the cloud provider to run instances in
	Zones []string `json:"zones"`
	// SSHPublicKeys is an optional slice of public keys to add to the
	// kubernetes node instances for hands on administration
	SSHPublicKeys []string `json:"sshPublicKeys"`
}
