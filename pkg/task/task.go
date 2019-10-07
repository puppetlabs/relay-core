package task

import (
	"crypto/sha1"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

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

// Metadata represents task metadata (such as the hash uniquely identifying the
// task).
type Metadata struct {
	Hash [sha1.Size]byte
}
