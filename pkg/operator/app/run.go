package app

import (
	"context"
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/handler/condition"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConfigureRun(ctx context.Context, rd *RunDeps, pr *obj.PipelineRun) {
	ConfigureRunStatus(rd.Run, pr)
	ConfigureRunStepStatus(ctx, rd, pr)
}

func ConfigureRunStatus(r *obj.Run, pr *obj.PipelineRun) {
	r.Object.Status.ObservedGeneration = r.Object.GetGeneration()

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

	for runConditionType, runCondition := range conds {
		UpdateStatusConditionIfTransitioned(runCondition, func() relayv1beta1.Condition {
			return condition.RunConditionHandlers[runConditionType](r, pr)
		})

	}

	runConditions := make([]relayv1beta1.RunCondition, 0)
	for runConditionType, condition := range conds {
		if !isConditionEmpty(condition) {
			runConditions = append(runConditions, relayv1beta1.RunCondition{
				Condition: *condition,
				Type:      runConditionType,
			})
		}
	}

	r.Object.Status.Conditions = runConditions
}

func ConfigureRunStepStatus(ctx context.Context, rd *RunDeps, pr *obj.PipelineRun) {
	wr := rd.Run
	wf := rd.Workflow

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
		if !ok || status == nil {
			continue
		}

		steps = append(steps,
			ConfigureStepStatus(ctx, rd, step.Name, action,
				status, currentStepStatus[step.Name]))
	}

	wr.Object.Status.Steps = steps
}

func ConfigureStepStatus(ctx context.Context, rd *RunDeps, stepName string, action *model.Step,
	status *tektonv1beta1.PipelineRunTaskRunStatus, currentStepStatus *relayv1beta1.StepStatus) *relayv1beta1.StepStatus {

	configMap := configmap.NewLocalConfigMap(rd.MutableConfigMap.Object)

	step := &relayv1beta1.StepStatus{
		Name: stepName,
	}

	if status.Status != nil {
		step.StartTime = status.Status.StartTime
		step.CompletionTime = status.Status.CompletionTime
	}

	step.Logs = configureStepLogs(status, currentStepStatus)

	actionStatus, _ := configmap.NewActionStatusManager(action, configMap).Get(ctx, action)

	step.Conditions = ConfigureRunStepStatusConditions(ctx, status, currentStepStatus, actionStatus)

	if timer, err := configmap.NewTimerManager(action, configMap).Get(ctx, model.TimerStepInit); err == nil {
		step.InitializationTime = &metav1.Time{Time: timer.Time}
	}

	step.Messages = make([]*relayv1beta1.StepMessage, 0)
	if messages, err := configmap.NewStepMessageManager(action, configMap).List(ctx); err == nil {
		for _, message := range messages {
			stepMessage := &relayv1beta1.StepMessage{
				Details:         message.Details,
				ObservationTime: metav1.Time{Time: message.Time},
				Severity:        relayv1beta1.StepMessageSeverityError,
			}

			if message.ConditionEvaluationResult != nil {
				expression := relayv1beta1.AsUnstructured(message.ConditionEvaluationResult.Expression)
				stepMessage.Source.WhenEvaluation = &relayv1beta1.WhenEvaluationStepMessageSource{
					Expression: &expression,
				}
			}

			if message.SchemaValidationResult != nil {
				expression := relayv1beta1.AsUnstructured(message.SchemaValidationResult.Expression)
				stepMessage.Source.SpecValidation = &relayv1beta1.SpecValidationStepMessageSource{
					Expression: &expression,
				}
			}

			step.Messages = append(step.Messages, stepMessage)
		}
	}

	step.Outputs = make([]*relayv1beta1.StepOutput, 0)
	if outputs, err := configmap.NewStepOutputManager(action, configMap).ListSelf(ctx); err == nil {
		for _, output := range outputs {
			sensitive := false
			if output.Metadata != nil {
				sensitive = output.Metadata.Sensitive
			}

			stepOutput := &relayv1beta1.StepOutput{
				Name:      output.Name,
				Sensitive: sensitive,
			}

			if !sensitive {
				value := relayv1beta1.AsUnstructured(output.Value)
				stepOutput.Value = &value
			}

			step.Outputs = append(step.Outputs, stepOutput)
		}
	}

	decs := []*relayv1beta1.Decorator{}
	if sdecs, err := configmap.NewStepDecoratorManager(action, configMap).List(ctx); err == nil {
		for _, sdec := range sdecs {
			decs = append(decs, &sdec.Value)
		}
	}

	step.Decorators = decs

	return step
}

func ConfigureRunStepStatusConditions(ctx context.Context,
	status *tektonv1beta1.PipelineRunTaskRunStatus, currentStepStatus *relayv1beta1.StepStatus,
	actionStatus *model.ActionStatus) []relayv1beta1.StepCondition {
	conds := map[relayv1beta1.StepConditionType]*relayv1beta1.Condition{
		relayv1beta1.StepCompleted: {},
		relayv1beta1.StepSkipped:   {},
		relayv1beta1.StepSucceeded: {},
	}

	if currentStepStatus != nil {
		for _, cond := range currentStepStatus.Conditions {
			if target, ok := conds[cond.Type]; ok {
				*target = cond.Condition
			}
		}
	}

	for stepConditionType, stepCondition := range conds {
		UpdateStatusConditionIfTransitioned(stepCondition, func() relayv1beta1.Condition {
			return condition.StepConditionHandlers[stepConditionType](status, actionStatus)
		})
	}

	stepConditions := make([]relayv1beta1.StepCondition, 0)
	for stepConditionType, condition := range conds {
		if !isConditionEmpty(condition) {
			stepConditions = append(stepConditions, relayv1beta1.StepCondition{
				Condition: *condition,
				Type:      stepConditionType,
			})
		}
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

func isConditionEmpty(cond *relayv1beta1.Condition) bool {
	return cond == nil ||
		(cond.Status == corev1.ConditionUnknown &&
			cond.Reason == "" && cond.Message == "")
}

// FIXME Temporary handling for legacy logs
func configureStepLogs(status *tektonv1beta1.PipelineRunTaskRunStatus, currentStepStatus *relayv1beta1.StepStatus) []*relayv1beta1.Log {
	if status.Status == nil || status.Status.PodName == "" {
		return nil
	}

	// The context needs to be preserved here (set elsewhere when logs are uploaded)
	// This coordination is messy, but necessary until the log handling is refactored...
	logContext := ""
	if currentStepStatus != nil {
		if len(currentStepStatus.Logs) > 0 {
			logContext = currentStepStatus.Logs[0].Context
		}
	}

	return []*relayv1beta1.Log{
		{
			Name:    status.Status.PodName,
			Context: logContext,
		},
	}
}
