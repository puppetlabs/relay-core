package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	PipelineRunKind = tektonv1beta1.SchemeGroupVersion.WithKind("PipelineRun")
)

type PipelineRun struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *tektonv1beta1.PipelineRun
}

func makePipelineRun(key client.ObjectKey, obj *tektonv1beta1.PipelineRun) *PipelineRun {
	pr := &PipelineRun{Key: key, Object: obj}
	pr.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&pr.Key, lifecycle.TypedObject{GVK: PipelineRunKind, Object: pr.Object})
	return pr
}

func (pr *PipelineRun) Copy() *PipelineRun {
	return makePipelineRun(pr.Key, pr.Object.DeepCopy())
}

func (pr *PipelineRun) Complete() bool {
	if !pr.Object.IsDone() && !pr.Object.IsCancelled() && !pr.Object.IsTimedOut() {
		return false
	}

	for _, tr := range pr.Object.Status.TaskRuns {
		if tr.Status == nil {
			continue
		}

		if WorkflowRunStatusFromCondition(tr.Status.Status) == WorkflowRunStatusInProgress {
			return false
		}
	}

	return true
}

func NewPipelineRun(key client.ObjectKey) *PipelineRun {
	return makePipelineRun(key, &tektonv1beta1.PipelineRun{})
}

func NewPipelineRunFromObject(obj *tektonv1beta1.PipelineRun) *PipelineRun {
	return makePipelineRun(client.ObjectKeyFromObject(obj), obj)
}

func NewPipelineRunPatcher(upd, orig *PipelineRun) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
