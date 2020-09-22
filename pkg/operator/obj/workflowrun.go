package obj

import (
	"context"
	"time"

	"github.com/puppetlabs/horsehead/v2/datastructure"
	"github.com/puppetlabs/horsehead/v2/graph"
	"github.com/puppetlabs/horsehead/v2/graph/traverse"
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

	WorkflowRunStatusPending    WorkflowRunStatus = "pending"
	WorkflowRunStatusInProgress WorkflowRunStatus = "in-progress"
	WorkflowRunStatusSuccess    WorkflowRunStatus = "success"
	WorkflowRunStatusFailure    WorkflowRunStatus = "failure"
	WorkflowRunStatusCancelled  WorkflowRunStatus = "cancelled"
	WorkflowRunStatusSkipped    WorkflowRunStatus = "skipped"
	WorkflowRunStatusTimedOut   WorkflowRunStatus = "timed-out"
)

var (
	WorkflowRunKind = nebulav1.SchemeGroupVersion.WithKind("WorkflowRun")
)

type WorkflowRun struct {
	Key    client.ObjectKey
	Object *nebulav1.WorkflowRun
}

var _ Loader = &WorkflowRun{}

func (wr *WorkflowRun) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, wr.Object)
}

func (wr *WorkflowRun) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, wr.Key, wr.Object)
}

func (wr *WorkflowRun) Own(ctx context.Context, other Ownable) error {
	return other.Owned(ctx, Owner{GVK: WorkflowRunKind, Object: wr.Object})
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

func (wr *WorkflowRun) Complete(ctx context.Context, cl client.Client) error {
	if wr.Object.Status.StartTime == nil {
		wr.Object.Status.StartTime = &metav1.Time{Time: time.Now()}
	}

	if wr.Object.Status.CompletionTime == nil {
		wr.Object.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	}

	wr.Object.Status.Status = string(WorkflowRunStatusSuccess)

	return wr.PersistStatus(ctx, cl)
}

func NewWorkflowRun(key client.ObjectKey) *WorkflowRun {
	return &WorkflowRun{
		Key:    key,
		Object: &nebulav1.WorkflowRun{},
	}
}

func taskRunConditionStatusSummary(status *tektonv1beta1.PipelineRunTaskRunStatus, name string) (sum nebulav1.WorkflowRunStatusSummary, ok bool) {
	for _, cond := range status.ConditionChecks {
		if cond.Status == nil {
			continue
		}

		sum.Name = name
		sum.Status = string(workflowRunStatus(cond.Status.Status))

		if cond.Status.StartTime != nil {
			sum.StartTime = cond.Status.StartTime
		}

		if cond.Status.CompletionTime != nil {
			sum.CompletionTime = cond.Status.CompletionTime
		}

		ok = true
		return
	}

	return
}

func taskRunStepStatusSummary(status *tektonv1beta1.PipelineRunTaskRunStatus, name string) (sum nebulav1.WorkflowRunStatusSummary, ok bool) {
	if status.Status == nil {
		return
	}

	sum.Name = name
	sum.Status = string(workflowRunStatus(status.Status.Status))

	if status.Status.StartTime != nil {
		sum.StartTime = status.Status.StartTime
	}

	if status.Status.CompletionTime != nil {
		sum.CompletionTime = status.Status.CompletionTime
	}

	ok = true
	return
}

func workflowRunSkipsPendingSteps(wr *WorkflowRun) bool {
	switch wr.Object.Status.Status {
	case string(WorkflowRunStatusCancelled), string(WorkflowRunStatusFailure), string(WorkflowRunStatusTimedOut):
		return true
	}

	return false
}

type workflowRunStatusSummariesByTaskName struct {
	steps      map[string]nebulav1.WorkflowRunStatusSummary
	conditions map[string]nebulav1.WorkflowRunStatusSummary
}

func workflowRunStatusSummaries(wr *WorkflowRun, pr *PipelineRun) *workflowRunStatusSummariesByTaskName {
	m := &workflowRunStatusSummariesByTaskName{
		steps:      make(map[string]nebulav1.WorkflowRunStatusSummary),
		conditions: make(map[string]nebulav1.WorkflowRunStatusSummary),
	}

	for name, taskRun := range pr.Object.Status.TaskRuns {
		if cond, ok := taskRunConditionStatusSummary(taskRun, name); ok {
			m.conditions[taskRun.PipelineTaskName] = cond
		}

		if step, ok := taskRunStepStatusSummary(taskRun, name); ok {
			if step.Status == string(WorkflowRunStatusPending) && workflowRunSkipsPendingSteps(wr) {
				step.Status = string(WorkflowRunStatusSkipped)
			}

			m.steps[taskRun.PipelineTaskName] = step
		}
	}

	return m
}

func ConfigureWorkflowRun(wr *WorkflowRun, pr *PipelineRun) {
	if wr.IsCancelled() {
		wr.Object.Status.Status = string(WorkflowRunStatusCancelled)
	} else {
		wr.Object.Status.Status = string(workflowRunStatus(pr.Object.Status.Status))
	}

	if then := pr.Object.Status.StartTime; then != nil {
		wr.Object.Status.StartTime = then
	}

	if then := pr.Object.Status.CompletionTime; then != nil {
		wr.Object.Status.CompletionTime = then
	}

	if wr.Object.Status.Steps == nil {
		wr.Object.Status.Steps = make(map[string]nebulav1.WorkflowRunStatusSummary)
	}

	if wr.Object.Status.Conditions == nil {
		wr.Object.Status.Conditions = make(map[string]nebulav1.WorkflowRunStatusSummary)
	}

	// These are status information organized by task name since we don't yet
	// have the step names.
	summariesByTaskName := workflowRunStatusSummaries(wr, pr)

	// This lets us mark pending steps as skipped if they won't ever be run.
	skipFinder := graph.NewSimpleDirectedGraphWithFeatures(graph.DeterministicIteration)

	for _, step := range wr.Object.Spec.Workflow.Steps {
		skipFinder.AddVertex(step.Name)
		for _, dep := range step.DependsOn {
			skipFinder.AddVertex(dep)
			skipFinder.Connect(dep, step.Name)
		}

		taskName := ModelStep(wr, step).Hash().HexEncoding()

		stepSummary, found := summariesByTaskName.steps[taskName]
		if !found {
			stepSummary.Status = string(WorkflowRunStatusPending)
		}

		// Retain any existing log record.
		if wr.Object.Status.Steps[step.Name].LogKey != "" {
			stepSummary.LogKey = wr.Object.Status.Steps[step.Name].LogKey
		}

		wr.Object.Status.Steps[step.Name] = stepSummary

		if conditionSummary, found := summariesByTaskName.conditions[taskName]; found {
			wr.Object.Status.Conditions[step.Name] = conditionSummary
		}
	}

	// Mark skipped in order.
	traverse.NewTopologicalOrderTraverser(skipFinder).ForEach(func(next graph.Vertex) error {
		self := wr.Object.Status.Steps[next.(string)]
		if self.Status != string(WorkflowRunStatusPending) {
			return nil
		}

		incoming, _ := skipFinder.IncomingEdgesOf(next)
		incoming.ForEach(func(edge graph.Edge) error {
			prev, _ := graph.OppositeVertexOf(skipFinder, edge, next)
			dependent := wr.Object.Status.Steps[prev.(string)]

			switch dependent.Status {
			case string(WorkflowRunStatusSkipped), string(WorkflowRunStatusFailure):
				self.Status = string(WorkflowRunStatusSkipped)
				wr.Object.Status.Steps[next.(string)] = self

				return datastructure.ErrStopIteration
			}

			return nil
		})

		return nil
	})
}

func workflowRunStatus(status duckv1beta1.Status) WorkflowRunStatus {
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
		if cs.Reason == resources.ReasonTimedOut {
			return WorkflowRunStatusTimedOut
		}
		return WorkflowRunStatusFailure
	default:
		return WorkflowRunStatusPending
	}
}
