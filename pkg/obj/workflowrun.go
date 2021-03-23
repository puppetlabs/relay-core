package obj

import (
	"context"
	"time"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/model"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowRunStatus string

const (
	WorkflowRunStateCancel = "cancel"

	WorkflowRunStatusQueued     WorkflowRunStatus = "queued"
	WorkflowRunStatusPending    WorkflowRunStatus = "pending"
	WorkflowRunStatusInProgress WorkflowRunStatus = "in-progress"
	WorkflowRunStatusSuccess    WorkflowRunStatus = "success"
	WorkflowRunStatusFailure    WorkflowRunStatus = "failure"
	WorkflowRunStatusCancelled  WorkflowRunStatus = "cancelled"
	WorkflowRunStatusSkipped    WorkflowRunStatus = "skipped"
	WorkflowRunStatusTimedOut   WorkflowRunStatus = "timed-out"
)

type WorkflowRun struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *nebulav1.WorkflowRun
}

func makeWorkflowRun(key client.ObjectKey, obj *nebulav1.WorkflowRun) *WorkflowRun {
	wr := &WorkflowRun{Key: key, Object: obj}
	wr.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&wr.Key, lifecycle.TypedObject{GVK: nebulav1.WorkflowRunKind, Object: wr.Object})
	return wr
}

func (wr *WorkflowRun) Copy() *WorkflowRun {
	return makeWorkflowRun(wr.Key, wr.Object.DeepCopy())
}

func (wr *WorkflowRun) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, wr.Object)
}

func (wr *WorkflowRun) PodSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			model.RelayControllerWorkflowRunIDLabel: wr.Key.Name,
		},
	}
}

func (wr *WorkflowRun) IsCancelled() bool {
	state, found := wr.Object.State.Workflow[WorkflowRunStateCancel]
	if !found {
		return false
	}

	return state.Value() == true
}

func (wr *WorkflowRun) Fail(ctx context.Context, cl client.Client) error {
	return wr.applyStatus(ctx, cl, WorkflowRunStatusFailure)
}

func (wr *WorkflowRun) Complete(ctx context.Context, cl client.Client) error {
	return wr.applyStatus(ctx, cl, WorkflowRunStatusSuccess)
}

func (wr *WorkflowRun) applyStatus(ctx context.Context, cl client.Client, status WorkflowRunStatus) error {
	if wr.Object.Status.StartTime == nil {
		wr.Object.Status.StartTime = &metav1.Time{Time: time.Now()}
	}

	if wr.Object.Status.CompletionTime == nil {
		wr.Object.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	}

	wr.Object.Status.Status = string(status)

	return wr.PersistStatus(ctx, cl)
}

func NewWorkflowRun(key client.ObjectKey) *WorkflowRun {
	return makeWorkflowRun(key, &nebulav1.WorkflowRun{})
}

func NewWorkflowRunFromObject(obj *nebulav1.WorkflowRun) *WorkflowRun {
	return makeWorkflowRun(client.ObjectKeyFromObject(obj), obj)
}

func NewWorkflowRunPatcher(upd, orig *WorkflowRun) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}

// TODO: Where does this method really belong?

func WorkflowRunStatusFromCondition(status duckv1beta1.Status) WorkflowRunStatus {
	cs := status.GetCondition(apis.ConditionSucceeded)
	if cs == nil {
		return WorkflowRunStatusPending
	}

	switch cs.Status {
	case corev1.ConditionUnknown:
		return WorkflowRunStatusInProgress
	case corev1.ConditionTrue:
		return WorkflowRunStatusSuccess
	case corev1.ConditionFalse:
		if cs.Reason == resources.ReasonConditionCheckFailed {
			return WorkflowRunStatusSkipped
		}
		if cs.Reason == tektonv1beta1.PipelineRunReasonTimedOut.String() {
			return WorkflowRunStatusTimedOut
		}
		return WorkflowRunStatusFailure
	default:
		return WorkflowRunStatusPending
	}
}
