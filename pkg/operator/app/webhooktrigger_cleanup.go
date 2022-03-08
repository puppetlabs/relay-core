package app

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WebhookTriggerCleanup automatically prunes parts of a webhook trigger that
// are no longer needed.
type WebhookTriggerCleanup struct {
	Deps           *WebhookTriggerDeps
	KnativeService *obj.KnativeService

	// KnativeRevisions are all of the revisions for the given service, which
	// can be matched up to stale resources.
	KnativeRevisions *KnativeRevisionSet
}

var (
	_ lifecycle.Deleter = &WebhookTriggerCleanup{}
	_ lifecycle.Loader  = &WebhookTriggerCleanup{}
)

func (wtc *WebhookTriggerCleanup) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	return false, nil
}

func (wtc *WebhookTriggerCleanup) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.Loaders{
		wtc.KnativeRevisions,
	}.Load(ctx, cl)
}

func NewWebhookTriggerCleanup(deps *WebhookTriggerDeps, ks *obj.KnativeService) *WebhookTriggerCleanup {
	return &WebhookTriggerCleanup{
		Deps:           deps,
		KnativeService: ks,

		KnativeRevisions: NewKnativeRevisionSetForKnativeService(ks),
	}
}

func ApplyWebhookTriggerCleanup(ctx context.Context, cl client.Client, deps *WebhookTriggerDeps, ksr *KnativeServiceResult) error {
	if ksr.Error != nil {
		return errmark.MarkTransient(fmt.Errorf("waiting for Knative service"))
	}

	wtc := NewWebhookTriggerCleanup(deps, ksr.KnativeService)
	if _, err := (lifecycle.RequiredLoader{Loader: wtc}).Load(ctx, cl); err != nil {
		return err
	}

	if _, err := wtc.Delete(ctx, cl); err != nil {
		return err
	}

	return nil
}
