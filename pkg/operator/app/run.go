package app

import (
	"context"
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func ConfigureRun(ctx context.Context, rd *RunDeps, pr *obj.PipelineRun) {
	ConfigureRunStatus(rd.Run, pr)
	ConfigureRunStepStatus(ctx, rd, pr)
}

func ConfigureRunStatus(r *obj.Run, pr *obj.PipelineRun) {
	if then := pr.Object.Status.StartTime; then != nil {
		r.Object.Status.StartTime = then
	}

	if then := pr.Object.Status.CompletionTime; then != nil {
		r.Object.Status.CompletionTime = then
	}

	conds := map[relayv1beta1.RunConditionType]*relayv1beta1.Condition{
		relayv1beta1.RunCancelled: {},
		relayv1beta1.RunCompleted: {},
		relayv1beta1.RunSucceeded: {},
		relayv1beta1.RunTimedOut:  {},
	}

	for _, cond := range r.Object.Status.Conditions {
		if target, ok := conds[cond.Type]; ok {
			*target = cond.Condition
		}
	}

	cs := pr.Object.Status.Status.GetCondition(apis.ConditionSucceeded)

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.RunCancelled], func() relayv1beta1.Condition {
		if r.IsCancelled() {
			return relayv1beta1.Condition{
				Status: corev1.ConditionTrue,
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.RunCompleted], func() relayv1beta1.Condition {
		if cs != nil {
			switch cs.Status {
			case corev1.ConditionTrue, corev1.ConditionFalse:
				return relayv1beta1.Condition{
					Status: corev1.ConditionTrue,
				}
			}

			return relayv1beta1.Condition{
				Status: corev1.ConditionFalse,
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.RunSucceeded], func() relayv1beta1.Condition {
		if cs != nil {
			switch cs.Status {
			case corev1.ConditionTrue, corev1.ConditionFalse:
				return relayv1beta1.Condition{
					Status: cs.Status,
				}
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.RunTimedOut], func() relayv1beta1.Condition {
		if cs != nil {
			switch cs.Status {
			case corev1.ConditionFalse:
				if cs.Reason == tektonv1beta1.PipelineRunReasonTimedOut.String() {
					return relayv1beta1.Condition{
						Status: corev1.ConditionTrue,
					}
				}
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	r.Object.Status.ObservedGeneration = r.Object.GetGeneration()
	r.Object.Status.Conditions = []relayv1beta1.RunCondition{
		{
			Condition: *conds[relayv1beta1.RunCancelled],
			Type:      relayv1beta1.RunCancelled,
		},
		{
			Condition: *conds[relayv1beta1.RunCompleted],
			Type:      relayv1beta1.RunCompleted,
		},
		{
			Condition: *conds[relayv1beta1.RunSucceeded],
			Type:      relayv1beta1.RunSucceeded,
		},
		{
			Condition: *conds[relayv1beta1.RunTimedOut],
			Type:      relayv1beta1.RunTimedOut,
		},
	}
}

func ConfigureRunStepStatus(ctx context.Context, rd *RunDeps, pr *obj.PipelineRun) {
	wr := rd.Run
	wf := rd.Workflow

	configMap := configmap.NewLocalConfigMap(rd.MutableConfigMap.Object)

	currentStepStatus := make(map[string]*relayv1beta1.StepStatus)

	for _, ss := range wr.Object.Status.Steps {
		currentStepStatus[ss.Name] = ss
	}

	statusByTaskName := make(map[string]*tektonv1beta1.PipelineRunTaskRunStatus)

	for _, tr := range pr.Object.Status.TaskRuns {
		statusByTaskName[tr.PipelineTaskName] = tr
	}

	steps := make([]*relayv1beta1.StepStatus, 0)

	for _, step := range wf.Object.Spec.Steps {
		action := ModelStep(wr, step)
		taskName := action.Hash().HexEncoding()

		status, ok := statusByTaskName[taskName]
		if !ok || status == nil || status.Status == nil {
			continue
		}

		step := relayv1beta1.StepStatus{
			Name:           step.Name,
			StartTime:      status.Status.StartTime,
			CompletionTime: status.Status.CompletionTime,

			// FIXME Temporary handling for legacy logs
			Logs: []*relayv1beta1.Log{
				{
					Name: status.Status.PodName,
				},
			},
		}

		cc := []relayv1beta1.StepCondition{}

		if currentStepStatus != nil && currentStepStatus[step.Name] != nil {
			// FIXME Temporary handling for legacy logs
			if len(currentStepStatus[step.Name].Logs) > 0 {
				step.Logs = currentStepStatus[step.Name].Logs
			}

			cc = currentStepStatus[step.Name].Conditions
		}

		// FIXME Temporary handling for legacy logs
		if len(step.Logs) > 0 {
			// This should never toggle from a valid name to a blank one, but just in case...
			if status.Status.PodName != "" {
				// The log name is always the pod name currently for legacy logs
				// Always update as the pod name may be blank when first initialized
				step.Logs[0].Name = status.Status.PodName
			}
		}

		cs := status.Status.GetCondition(apis.ConditionSucceeded)

		step.Conditions = ConfigureRunStepStatusConditions(ctx, cs, cc)

		if timer, err := configmap.NewTimerManager(action, configMap).Get(ctx, model.TimerStepInit); err == nil {
			step.InitializationTime = &metav1.Time{Time: timer.Time}
		}

		step.Outputs = make([]*relayv1beta1.StepOutput, 0)
		if outputs, err := configmap.NewStepOutputManager(action, configMap).ListSelf(ctx); err == nil {
			for _, output := range outputs {
				value := relayv1beta1.AsUnstructured(output.Value)
				step.Outputs = append(step.Outputs,
					&relayv1beta1.StepOutput{
						Name:      output.Name,
						Value:     &value,
						Sensitive: false,
					})
			}
		}

		decs := []*relayv1beta1.Decorator{}
		if sdecs, err := configmap.NewStepDecoratorManager(action, configMap).List(ctx); err == nil {
			for _, sdec := range sdecs {
				decs = append(decs, &sdec.Value)
			}
		}

		step.Decorators = decs

		steps = append(steps, &step)
	}

	wr.Object.Status.Steps = steps
}

func ConfigureRunStepStatusConditions(ctx context.Context, condition *apis.Condition, currentConditions []relayv1beta1.StepCondition) []relayv1beta1.StepCondition {
	conds := map[relayv1beta1.StepConditionType]*relayv1beta1.Condition{
		relayv1beta1.StepCompleted: {},
		relayv1beta1.StepSkipped:   {},
		relayv1beta1.StepSucceeded: {},
	}

	for _, cond := range currentConditions {
		if target, ok := conds[cond.Type]; ok {
			*target = cond.Condition
		}
	}

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.StepCompleted], func() relayv1beta1.Condition {
		if condition != nil {
			switch condition.Status {
			case corev1.ConditionTrue, corev1.ConditionFalse:
				return relayv1beta1.Condition{
					Status: corev1.ConditionTrue,
				}
			case corev1.ConditionUnknown:
				return relayv1beta1.Condition{
					Status: corev1.ConditionFalse,
				}
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.StepSkipped], func() relayv1beta1.Condition {
		if condition != nil {
			switch condition.Status {
			case corev1.ConditionFalse:
				if condition.Reason == resources.ReasonConditionCheckFailed {
					return relayv1beta1.Condition{
						Status: corev1.ConditionTrue,
					}
				}
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.StepSucceeded], func() relayv1beta1.Condition {
		if condition != nil {
			switch condition.Status {
			case corev1.ConditionTrue, corev1.ConditionFalse:
				return relayv1beta1.Condition{
					Status: condition.Status,
				}
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	stepConditions := []relayv1beta1.StepCondition{
		{
			Condition: *conds[relayv1beta1.StepCompleted],
			Type:      relayv1beta1.StepCompleted,
		},
		{
			Condition: *conds[relayv1beta1.StepSkipped],
			Type:      relayv1beta1.StepSkipped,
		},
		{
			Condition: *conds[relayv1beta1.StepSucceeded],
			Type:      relayv1beta1.StepSucceeded,
		},
	}

	return stepConditions
}

func ConfigureRunWithSpecificStatus(r *obj.Run, rc relayv1beta1.RunConditionType, status corev1.ConditionStatus) {
	if r.Object.Status.StartTime == nil {
		r.Object.Status.StartTime = &metav1.Time{Time: time.Now()}
	}

	if r.Object.Status.CompletionTime == nil {
		r.Object.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	}

	conds := map[relayv1beta1.RunConditionType]*relayv1beta1.Condition{
		relayv1beta1.RunCancelled: {},
		relayv1beta1.RunCompleted: {},
		relayv1beta1.RunSucceeded: {},
		relayv1beta1.RunTimedOut:  {},
	}

	for _, cond := range r.Object.Status.Conditions {
		if target, ok := conds[cond.Type]; ok {
			*target = cond.Condition
		}
	}

	UpdateStatusConditionIfTransitioned(conds[rc], func() relayv1beta1.Condition {
		return relayv1beta1.Condition{
			Status: status,
		}
	})

	r.Object.Status.ObservedGeneration = r.Object.GetGeneration()
	r.Object.Status.Conditions = []relayv1beta1.RunCondition{
		{
			Condition: *conds[rc],
			Type:      rc,
		},
	}
}