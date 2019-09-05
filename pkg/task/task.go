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

type Metadata struct {
	ID   string
	Name string
}
