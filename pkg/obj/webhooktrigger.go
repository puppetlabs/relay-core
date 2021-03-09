package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
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

type WebhookTrigger struct {
	Key    client.ObjectKey
	Object *relayv1beta1.WebhookTrigger
}

var _ lifecycle.Finalizable = &WebhookTrigger{}
var _ lifecycle.Loader = &WebhookTrigger{}
var _ lifecycle.Persister = &WebhookTrigger{}

func (wt *WebhookTrigger) Finalizing() bool {
	return !wt.Object.GetDeletionTimestamp().IsZero()
}

func (wt *WebhookTrigger) AddFinalizer(ctx context.Context, name string) bool {
	return helper.AddFinalizer(&wt.Object.ObjectMeta, name)
}

func (wt *WebhookTrigger) RemoveFinalizer(ctx context.Context, name string) bool {
	return helper.RemoveFinalizer(&wt.Object.ObjectMeta, name)
}

func (wt *WebhookTrigger) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, wt.Key, wt.Object)
}

func (wt *WebhookTrigger) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, wt.Object, helper.WithObjectKey(wt.Key))
}

func (wt *WebhookTrigger) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, wt.Object)
}

func (wt *WebhookTrigger) Ready() bool {
	for _, cond := range wt.Object.Status.Conditions {
		if cond.Type != relayv1beta1.WebhookTriggerReady {
			continue
		}

		return cond.Status == corev1.ConditionTrue
	}

	return false
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
