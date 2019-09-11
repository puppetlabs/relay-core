package task

import "github.com/puppetlabs/nebula-tasks/pkg/taskutil"

const (
	DefaultName     = "default"
	DefaultPath     = "/workspace"
	DefaultRevision = "master"
	KubeConfigFile  = "kubeconfig"
)

type TaskInterface struct {
	opts taskutil.DefaultPlanOptions
}

func NewTaskInterface(opts taskutil.DefaultPlanOptions) *TaskInterface {
	return &TaskInterface{opts}
}

// Metadata represents task metadata (such as the name of the task and ID).
type Metadata struct {
	ID   string
	Name string
}
