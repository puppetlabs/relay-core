package models

const (
	ClusterStatusRunning = "running"
	ClusterStatusStopped = "stopped"
	ClusterStatusFailed  = "failed"
)

type K8sClusterState struct {
	Status string `json:"status"`
}
