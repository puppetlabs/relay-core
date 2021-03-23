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
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *relayv1beta1.WebhookTrigger
}

func makeWebhookTrigger(key client.ObjectKey, obj *relayv1beta1.WebhookTrigger) *WebhookTrigger {
	wt := &WebhookTrigger{Key: key, Object: obj}
	wt.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&wt.Key, lifecycle.TypedObject{GVK: relayv1beta1.WebhookTriggerKind, Object: wt.Object})
	return wt
}

func (wt *WebhookTrigger) Copy() *WebhookTrigger {
	return makeWebhookTrigger(wt.Key, wt.Object.DeepCopy())
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
	return makeWebhookTrigger(key, &relayv1beta1.WebhookTrigger{})
}

func NewWebhookTriggerFromObject(obj *relayv1beta1.WebhookTrigger) *WebhookTrigger {
	return makeWebhookTrigger(client.ObjectKeyFromObject(obj), obj)
}

func NewWebhookTriggerPatcher(upd, orig *WebhookTrigger) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
