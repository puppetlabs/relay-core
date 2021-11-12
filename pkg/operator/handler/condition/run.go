package condition

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

type RunConditionHandlerFunc func(r *obj.Run, pr *obj.PipelineRun) relayv1beta1.Condition

var (
	RunConditionHandlers = map[relayv1beta1.RunConditionType]RunConditionHandlerFunc{
		relayv1beta1.RunCancelled: runCancelledHandler,
		relayv1beta1.RunCompleted: runCompletedHandler,
		relayv1beta1.RunSucceeded: runSucceededHandler,
		relayv1beta1.RunTimedOut:  runTimedOutHandler,
	}
)

var runCancelledHandler = RunConditionHandlerFunc(func(r *obj.Run, pr *obj.PipelineRun) relayv1beta1.Condition {
	if r.IsCancelled() {
		return relayv1beta1.Condition{
			Status: corev1.ConditionTrue,
		}
	}

	return relayv1beta1.Condition{
		Status: corev1.ConditionUnknown,
	}
})

var runCompletedHandler = RunConditionHandlerFunc(func(r *obj.Run, pr *obj.PipelineRun) relayv1beta1.Condition {
	cs := pr.Object.Status.Status.GetCondition(apis.ConditionSucceeded)

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

var runSucceededHandler = RunConditionHandlerFunc(func(r *obj.Run, pr *obj.PipelineRun) relayv1beta1.Condition {
	cs := pr.Object.Status.Status.GetCondition(apis.ConditionSucceeded)

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

var runTimedOutHandler = RunConditionHandlerFunc(func(r *obj.Run, pr *obj.PipelineRun) relayv1beta1.Condition {
	cs := pr.Object.Status.Status.GetCondition(apis.ConditionSucceeded)

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
