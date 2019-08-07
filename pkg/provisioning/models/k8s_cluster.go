package models

type ClusterStatus string

func (cs ClusterStatus) String() string {
	return string(cs)
}

const (
	ClusterStatusReady    ClusterStatus = "Ready"
	ClusterStatusNotReady ClusterStatus = "NotReady"
)

type K8sCluster struct {
	Name   string        `json:"name"`
	Status ClusterStatus `json:"status"`
}
