package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WebhookTriggerStatusReasonServiceReady = "ServiceReady"
	WebhookTriggerStatusReasonServiceError = "ServiceError"

	WebhookTriggerStatusReasonReady = "Ready"
	WebhookTriggerStatusReasonError = "Error"
)

var (
	WebhookTriggerKind = relayv1beta1.SchemeGroupVersion.WithKind("WebhookTrigger")
)

type WebhookTrigger struct {
	Key    client.ObjectKey
	Object *relayv1beta1.WebhookTrigger
}

var _ Loader = &WebhookTrigger{}

func (wt *WebhookTrigger) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, wt.Object)
}

func (wt *WebhookTrigger) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, wt.Key, wt.Object)
}

func (wt *WebhookTrigger) Own(ctx context.Context, other Ownable) error {
	return other.Owned(ctx, Owner{GVK: WebhookTriggerKind, Object: wt.Object})
}

func (wt *WebhookTrigger) PodSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			model.RelayControllerWebhookTriggerIDLabel: wt.Key.Name,
		},
	}
}

func NewWebhookTrigger(key client.ObjectKey) *WebhookTrigger {
	return &WebhookTrigger{
		Key:    key,
		Object: &relayv1beta1.WebhookTrigger{},
	}
}

func ConfigureWebhookTrigger(wt *WebhookTrigger, ksr *KnativeServiceResult) {
	// Set up our initial map from the existing data.
	conds := map[relayv1beta1.WebhookTriggerConditionType]*relayv1beta1.Condition{
		relayv1beta1.WebhookTriggerServiceReady: &relayv1beta1.Condition{},
		relayv1beta1.WebhookTriggerReady:        &relayv1beta1.Condition{},
	}

	for _, cond := range wt.Object.Status.Conditions {
		conds[cond.Type] = &cond.Condition
	}

	// Update with data from Knative service.
	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.WebhookTriggerServiceReady], func() relayv1beta1.Condition {
		if ksr.Error != nil {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  WebhookTriggerStatusReasonServiceError,
				Message: ksr.Error.Error(),
			}
		} else if ksr.KnativeService != nil && ksr.KnativeService.Object.Status.IsReady() {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  WebhookTriggerStatusReasonServiceReady,
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
				Reason:  WebhookTriggerStatusReasonReady,
				Message: "The webhook trigger is configured.",
			}
		case corev1.ConditionFalse:
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  WebhookTriggerStatusReasonError,
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

	if ksr.KnativeService != nil && ksr.KnativeService.Object.Status.IsReady() && ksr.KnativeService.Object.Status.URL != nil {
		wt.Object.Status.URL = ksr.KnativeService.Object.Status.URL.String()
	}
}
