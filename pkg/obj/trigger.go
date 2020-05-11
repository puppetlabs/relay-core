package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
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

var _ Persister = &WebhookTrigger{}
var _ Loader = &WebhookTrigger{}

func (wt *WebhookTrigger) Persist(ctx context.Context, cl client.Client) error {
	if err := CreateOrUpdate(ctx, cl, wt.Key, wt.Object); err != nil {
		return err
	}

	if err := cl.Status().Update(ctx, wt.Object); err != nil {
		return err
	}

	return nil
}

func (wt *WebhookTrigger) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, wt.Key, wt.Object)
}

func (wt *WebhookTrigger) Own(ctx context.Context, other Ownable) {
	other.Owned(ctx, metav1.NewControllerRef(wt.Object, WebhookTriggerKind))
}

func NewWebhookTrigger(key client.ObjectKey) *WebhookTrigger {
	return &WebhookTrigger{
		Key:    key,
		Object: &relayv1beta1.WebhookTrigger{},
	}
}
