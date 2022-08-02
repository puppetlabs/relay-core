package condition

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

type RunConditionHandlerFunc func(r *obj.Run) relayv1beta1.Condition

var (
	RunConditionHandlers = map[relayv1beta1.RunConditionType]RunConditionHandlerFunc{
		relayv1beta1.RunCancelled: runCancelledHandler,
		relayv1beta1.RunCompleted: runCompletedHandler,
		relayv1beta1.RunSucceeded: runSucceededHandler,
	}
)

var runCancelledHandler = RunConditionHandlerFunc(func(r *obj.Run) relayv1beta1.Condition {
	if r.IsCancelled() {
		return relayv1beta1.Condition{
			Status: corev1.ConditionTrue,
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})

var runCompletedHandler = RunConditionHandlerFunc(func(r *obj.Run) relayv1beta1.Condition {
	for _, step := range r.Object.Status.Steps {
		for _, condition := range step.Conditions {
			switch condition.Type {
			case relayv1beta1.StepCompleted:
				if condition.Status != corev1.ConditionTrue {
					return relayv1beta1.Condition{
						Status: corev1.ConditionFalse,
					}
				}
			}
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionTrue,
	}
})

var runSucceededHandler = RunConditionHandlerFunc(func(r *obj.Run) relayv1beta1.Condition {
	status := corev1.ConditionTrue
	for _, step := range r.Object.Status.Steps {
		for _, condition := range step.Conditions {
			switch condition.Type {
			case relayv1beta1.StepSucceeded:
				switch condition.Status {
				case corev1.ConditionFalse:
					return relayv1beta1.Condition{
						Status: corev1.ConditionFalse,
					}
				case corev1.ConditionUnknown:
					status = corev1.ConditionUnknown
				}
			}
		}
	}

	return relayv1beta1.Condition{
		Status: status,
	}
})
