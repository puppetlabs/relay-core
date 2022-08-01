package condition

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	corev1 "k8s.io/api/core/v1"
)

type StepConditionHandlerFunc func(actionStatus *model.ActionStatus) relayv1beta1.Condition

var (
	StepConditionHandlers = map[relayv1beta1.StepConditionType]StepConditionHandlerFunc{
		relayv1beta1.StepCompleted: stepCompletedHandler,
		relayv1beta1.StepSkipped:   stepSkippedHandler,
		relayv1beta1.StepSucceeded: stepSucceededHandler,
	}
)

var stepCompletedHandler = StepConditionHandlerFunc(func(actionStatus *model.ActionStatus) relayv1beta1.Condition {
	if actionStatus != nil {
		if actionStatus.ProcessState != nil {
			return relayv1beta1.Condition{
				Status: corev1.ConditionTrue,
			}
		} else if actionStatus.WhenCondition != nil &&
			actionStatus.WhenCondition.WhenConditionStatus == model.WhenConditionStatusSatisfied {
			return relayv1beta1.Condition{
				Status: corev1.ConditionFalse,
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})

var stepSkippedHandler = StepConditionHandlerFunc(func(actionStatus *model.ActionStatus) relayv1beta1.Condition {
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

var stepSucceededHandler = StepConditionHandlerFunc(func(actionStatus *model.ActionStatus) relayv1beta1.Condition {
	if actionStatus != nil {
		if succeeded, err := actionStatus.Succeeded(); err == nil {
			if succeeded {
				return relayv1beta1.Condition{
					Status: corev1.ConditionTrue,
				}
			}

			return relayv1beta1.Condition{
				Status: corev1.ConditionFalse,
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})
