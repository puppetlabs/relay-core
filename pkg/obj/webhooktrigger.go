package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
