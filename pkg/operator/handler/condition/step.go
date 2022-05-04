package condition

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

type StepConditionHandlerFunc func(status *tektonv1beta1.PipelineRunTaskRunStatus, actionStatus *model.ActionStatus) relayv1beta1.Condition

var (
	StepConditionHandlers = map[relayv1beta1.StepConditionType]StepConditionHandlerFunc{
		relayv1beta1.StepCompleted: stepCompletedHandler,
		relayv1beta1.StepSkipped:   stepSkippedHandler,
		relayv1beta1.StepSucceeded: stepSucceededHandler,
	}
)

var stepCompletedHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus, actionStatus *model.ActionStatus) relayv1beta1.Condition {
	stepStatus := &apis.Condition{}
	if status.Status != nil {
		stepStatus = status.Status.GetCondition(apis.ConditionSucceeded)
	}

	if stepStatus != nil {
		switch stepStatus.Status {
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

var stepSkippedHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus, actionStatus *model.ActionStatus) relayv1beta1.Condition {
	if actionStatus != nil && actionStatus.WhenCondition != nil {
		switch actionStatus.WhenCondition.WhenConditionStatus {
		case model.WhenConditionStatusEvaluating:
			return relayv1beta1.Condition{
				Status: corev1.ConditionUnknown,
				Reason: model.WhenConditionStatusEvaluating.String(),
			}
		case model.WhenConditionStatusFailure:
			return relayv1beta1.Condition{
				Status: corev1.ConditionTrue,
				Reason: model.WhenConditionStatusFailure.String(),
			}
		case model.WhenConditionStatusNotSatisfied:
			return relayv1beta1.Condition{
				Status: corev1.ConditionTrue,
				Reason: model.WhenConditionStatusNotSatisfied.String(),
			}
		case model.WhenConditionStatusSatisfied:
			return relayv1beta1.Condition{
				Status: corev1.ConditionFalse,
				Reason: model.WhenConditionStatusSatisfied.String(),
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
		Reason: model.WhenConditionStatusUnknown.String(),
	}
})

var stepSucceededHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus, actionStatus *model.ActionStatus) relayv1beta1.Condition {
	stepStatus := &apis.Condition{}
	if status.Status != nil {
		stepStatus = status.Status.GetCondition(apis.ConditionSucceeded)
	}

	if stepStatus != nil {
		switch stepStatus.Status {
		case corev1.ConditionTrue, corev1.ConditionFalse:
			return relayv1beta1.Condition{
				Status: stepStatus.Status,
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})
