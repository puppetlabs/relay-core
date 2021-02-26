package app

import (
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureWebhookTrigger(wt *obj.WebhookTrigger, ksr *KnativeServiceResult) {
	// Set up our initial map from the existing data.
	conds := map[relayv1beta1.WebhookTriggerConditionType]*relayv1beta1.Condition{
		relayv1beta1.WebhookTriggerServiceReady: &relayv1beta1.Condition{},
		relayv1beta1.WebhookTriggerReady:        &relayv1beta1.Condition{},
	}

	for _, cond := range wt.Object.Status.Conditions {
		*conds[cond.Type] = cond.Condition
	}

	// Update with data from Knative service.
	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.WebhookTriggerServiceReady], func() relayv1beta1.Condition {
		if ksr.Error != nil {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  obj.WebhookTriggerStatusReasonServiceError,
				Message: ksr.Error.Error(),
			}
		} else if ksr.KnativeService != nil && ksr.KnativeService.Object.IsReady() {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.WebhookTriggerStatusReasonServiceReady,
				Message: "The service is ready to handle requests.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.WebhookTriggerReady], func() relayv1beta1.Condition {
		switch AggregateStatusConditions(*conds[relayv1beta1.WebhookTriggerServiceReady]) {
		case corev1.ConditionTrue:
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.WebhookTriggerStatusReasonReady,
				Message: "The webhook trigger is configured.",
			}
		case corev1.ConditionFalse:
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  obj.WebhookTriggerStatusReasonError,
				Message: "One or more webhook trigger components failed.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	// Write back to status.
	wt.Object.Status = relayv1beta1.WebhookTriggerStatus{
		ObservedGeneration: wt.Object.GetGeneration(),
		Conditions: []relayv1beta1.WebhookTriggerCondition{
			{
				Condition: *conds[relayv1beta1.WebhookTriggerServiceReady],
				Type:      relayv1beta1.WebhookTriggerServiceReady,
			},
			{
				Condition: *conds[relayv1beta1.WebhookTriggerReady],
				Type:      relayv1beta1.WebhookTriggerReady,
			},
		},
	}

	if ksr.KnativeService != nil {
		wt.Object.Status.Namespace = ksr.KnativeService.Key.Namespace

		if ksr.KnativeService.Object.IsReady() && ksr.KnativeService.Object.Status.URL != nil {
			wt.Object.Status.URL = ksr.KnativeService.Object.Status.URL.String()
		} else {
			wt.Object.Status.URL = ""
		}
	} else {
		wt.Object.Status.Namespace = ""
		wt.Object.Status.URL = ""
	}
}
