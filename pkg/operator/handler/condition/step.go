package condition

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

const (
	StepSkippedReasonWhenConditionEvaluating   = "WhenConditionEvaluating"
	StepSkippedReasonWhenConditionFailure      = "WhenConditionFailure"
	StepSkippedReasonWhenConditionNotSatisfied = "WhenConditionNotSatisfied"
	StepSkippedReasonWhenConditionSatisfied    = "WhenConditionSatisfied"
)

type StepConditionHandlerFunc func(status *tektonv1beta1.PipelineRunTaskRunStatus) relayv1beta1.Condition

var (
	StepConditionHandlers = map[relayv1beta1.StepConditionType]StepConditionHandlerFunc{
		relayv1beta1.StepCompleted: stepCompletedHandler,
		relayv1beta1.StepSkipped:   stepSkippedHandler,
		relayv1beta1.StepSucceeded: stepSucceededHandler,
	}
)

var stepCompletedHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus) relayv1beta1.Condition {
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

var stepSkippedHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus) relayv1beta1.Condition {
	for _, cond := range status.ConditionChecks {
		if cond == nil || cond.Status == nil {
			continue
		}

		conditionStatus := cond.Status.GetCondition(apis.ConditionSucceeded)

		if conditionStatus != nil {
			switch conditionStatus.Status {
			case corev1.ConditionFalse:
				if cond.Status.Check.Terminated != nil {
					if cond.Status.Check.Terminated.ExitCode >= 2 {
						return relayv1beta1.Condition{
							Status: corev1.ConditionTrue,
							Reason: StepSkippedReasonWhenConditionFailure,
						}
					}

					return relayv1beta1.Condition{
						Status: corev1.ConditionTrue,
						Reason: StepSkippedReasonWhenConditionNotSatisfied,
					}
				}

				return relayv1beta1.Condition{
					Status: corev1.ConditionTrue,
				}
			case corev1.ConditionTrue:
				return relayv1beta1.Condition{
					Status: corev1.ConditionFalse,
					Reason: StepSkippedReasonWhenConditionSatisfied,
				}
			case corev1.ConditionUnknown:
				return relayv1beta1.Condition{
					Status: corev1.ConditionUnknown,
					Reason: StepSkippedReasonWhenConditionEvaluating,
				}
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})

var stepSucceededHandler = StepConditionHandlerFunc(func(status *tektonv1beta1.PipelineRunTaskRunStatus) relayv1beta1.Condition {
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
