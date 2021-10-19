package app

import (
	"context"

	"github.com/puppetlabs/leg/datastructure"
	"github.com/puppetlabs/leg/graph"
	"github.com/puppetlabs/leg/graph/traverse"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func taskRunConditionStatusSummary(status *tektonv1beta1.PipelineRunTaskRunStatus, name string) (sum nebulav1.WorkflowRunStatusSummary, ok bool) {
	for _, cond := range status.ConditionChecks {
		if cond.Status == nil {
			continue
		}

		sum.Name = name
		sum.Status = string(obj.WorkflowRunStatusFromCondition(cond.Status.Status))

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
	sum.Status = string(obj.WorkflowRunStatusFromCondition(status.Status.Status))

	if status.Status.StartTime != nil {
		sum.StartTime = status.Status.StartTime
	}

	if status.Status.CompletionTime != nil {
		sum.CompletionTime = status.Status.CompletionTime
	}

	ok = true
	return
}

func workflowRunSkipsPendingSteps(wr *obj.WorkflowRun) bool {
	switch wr.Object.Status.Status {
	case string(obj.WorkflowRunStatusCancelled), string(obj.WorkflowRunStatusFailure), string(obj.WorkflowRunStatusTimedOut):
		return true
	}

	return false
}

type workflowRunStatusSummariesByTaskName struct {
	steps      map[string]nebulav1.WorkflowRunStatusSummary
	conditions map[string]nebulav1.WorkflowRunStatusSummary
}

func workflowRunStatusSummaries(ctx context.Context, wr *obj.WorkflowRun, pr *obj.PipelineRun) *workflowRunStatusSummariesByTaskName {
	m := &workflowRunStatusSummariesByTaskName{
		steps:      make(map[string]nebulav1.WorkflowRunStatusSummary),
		conditions: make(map[string]nebulav1.WorkflowRunStatusSummary),
	}

	for name, taskRun := range pr.Object.Status.TaskRuns {
		if cond, ok := taskRunConditionStatusSummary(taskRun, name); ok {
			m.conditions[taskRun.PipelineTaskName] = cond
		}

		if step, ok := taskRunStepStatusSummary(taskRun, name); ok {
			if step.Status == string(obj.WorkflowRunStatusPending) && workflowRunSkipsPendingSteps(wr) {
				step.Status = string(obj.WorkflowRunStatusSkipped)
			}

			m.steps[taskRun.PipelineTaskName] = step
		}
	}

	return m
}

func ConfigureWorkflowRun(ctx context.Context, wrd *WorkflowRunDeps, pr *obj.PipelineRun) {
	wr := wrd.WorkflowRun

	if wr.IsCancelled() {
		wr.Object.Status.Status = string(obj.WorkflowRunStatusCancelled)
	} else {
		wr.Object.Status.Status = string(obj.WorkflowRunStatusFromCondition(pr.Object.Status.Status))
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
	summariesByTaskName := workflowRunStatusSummaries(ctx, wr, pr)

	// This lets us mark pending steps as skipped if they won't ever be run.
	skipFinder := graph.NewSimpleDirectedGraphWithFeatures(graph.DeterministicIteration)

	// This is the ConfigMap that holds internal, mutable data (outputs, timing, etc.)
	configMap := configmap.NewLocalConfigMap(wrd.MutableConfigMap.Object)

	for _, step := range wrd.Workflow.Object.Spec.Steps {
		skipFinder.AddVertex(step.Name)
		for _, dep := range step.DependsOn {
			skipFinder.AddVertex(dep)
			skipFinder.Connect(dep, step.Name)
		}

		action := ModelStep(wr, step)
		taskName := action.Hash().HexEncoding()

		stepSummary, found := summariesByTaskName.steps[taskName]
		if !found {
			stepSummary.Status = string(obj.WorkflowRunStatusPending)
		}

		if timer, err := configmap.NewTimerManager(action, configMap).Get(ctx, model.TimerStepInit); err == nil {
			stepSummary.InitTime = &metav1.Time{Time: timer.Time}
		}

		// Retain any existing log record.
		if wr.Object.Status.Steps[step.Name].LogKey != "" {
			stepSummary.LogKey = wr.Object.Status.Steps[step.Name].LogKey
		}

		stepSummary.Outputs = relayv1beta1.NewUnstructuredObject(nil)
		if outputs, err := configmap.NewStepOutputManager(action, configMap).ListSelf(ctx); err == nil {
			for _, output := range outputs {
				stepSummary.Outputs[output.Name] = relayv1beta1.AsUnstructured(output.Value)
			}
		}

		decs := []relayv1beta1.Decorator{}
		if sdecs, err := configmap.NewStepDecoratorManager(action, configMap).List(ctx); err == nil {
			for _, sdec := range sdecs {
				decs = append(decs, sdec.Value)
			}
		}

		stepSummary.Decorators = decs

		wr.Object.Status.Steps[step.Name] = stepSummary

		if conditionSummary, found := summariesByTaskName.conditions[taskName]; found {
			wr.Object.Status.Conditions[step.Name] = conditionSummary
		}
	}

	// Mark skipped in order.
	traverse.NewTopologicalOrderTraverser(skipFinder).ForEach(func(next graph.Vertex) error {
		self := wr.Object.Status.Steps[next.(string)]
		if self.Status != string(obj.WorkflowRunStatusPending) {
			return nil
		}

		incoming, _ := skipFinder.IncomingEdgesOf(next)
		incoming.ForEach(func(edge graph.Edge) error {
			prev, _ := graph.OppositeVertexOf(skipFinder, edge, next)
			dependent := wr.Object.Status.Steps[prev.(string)]

			switch dependent.Status {
			case string(obj.WorkflowRunStatusSkipped), string(obj.WorkflowRunStatusFailure):
				self.Status = string(obj.WorkflowRunStatusSkipped)
				wr.Object.Status.Steps[next.(string)] = self

				return datastructure.ErrStopIteration
			}

			return nil
		})

		return nil
	})
}
