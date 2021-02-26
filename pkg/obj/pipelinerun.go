package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipelineRun struct {
	Key    client.ObjectKey
	Object *tektonv1beta1.PipelineRun
}

var _ lifecycle.LabelAnnotatableFrom = &PipelineRun{}
var _ lifecycle.Loader = &PipelineRun{}
var _ lifecycle.Ownable = &PipelineRun{}
var _ lifecycle.Persister = &PipelineRun{}

func (pr *PipelineRun) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&pr.Object.ObjectMeta, from)
}

func (pr *PipelineRun) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, pr.Key, pr.Object)
}

func (pr *PipelineRun) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(pr.Object, owner)
}

func (pr *PipelineRun) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, pr.Object, helper.WithObjectKey(pr.Key))
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
	return &PipelineRun{
		Key:    key,
		Object: &tektonv1beta1.PipelineRun{},
	}
}
