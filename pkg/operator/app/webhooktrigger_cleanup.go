package app

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/admission"
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

	// For tool injection, we keep checkouts around as long as at least one
	// revision of our Knative service references it. The checkouts are keyed by
	// the pool version, and we always create a new revision with the latest
	// pool version, so this will always be eventually consistent.
	ToolInjectionCheckouts *CheckoutSet
}

var (
	_ lifecycle.Deleter = &WebhookTriggerCleanup{}
	_ lifecycle.Loader  = &WebhookTriggerCleanup{}
)

func (wtc *WebhookTriggerCleanup) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	var claimNames []string

	// Even if a revision hasn't been created yet, we want to retain the claim
	// that is explicitly specified on the current service.
	if claimName := wtc.KnativeService.Object.Spec.Template.GetLabels()[admission.ToolsVolumeClaimAnnotation]; claimName != "" {
		claimNames = append(claimNames, claimName)
	}

	// Then we load the names from the rest of the revisions.
	for _, rev := range wtc.KnativeRevisions.Revisions {
		if claimName := rev.Object.GetLabels()[admission.ToolsVolumeClaimAnnotation]; claimName != "" {
			claimNames = append(claimNames, claimName)
		}
	}

	// Filter out all of the checkouts that are currently referenced and delete
	// the rest.
	del := RemoveCheckoutsWithClaimNames(wtc.ToolInjectionCheckouts.Checkouts, claimNames)
	some := len(del) == 0

	for _, co := range del {
		if ok, err := co.Delete(ctx, cl, opts...); err != nil {
			return false, err
		} else if ok {
			some = true
		}
	}

	return some, nil
}

func (wtc *WebhookTriggerCleanup) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.Loaders{
		wtc.KnativeRevisions,
		wtc.ToolInjectionCheckouts,
	}.Load(ctx, cl)
}

func NewWebhookTriggerCleanup(deps *WebhookTriggerDeps, ks *obj.KnativeService) *WebhookTriggerCleanup {
	return &WebhookTriggerCleanup{
		Deps:           deps,
		KnativeService: ks,

		KnativeRevisions: NewKnativeRevisionSetForKnativeService(ks),
		ToolInjectionCheckouts: NewCheckoutSet(
			client.InNamespace(deps.TenantDeps.Namespace.Name),
			client.MatchingLabels{model.RelayControllerWebhookTriggerIDLabel: deps.WebhookTrigger.Key.Name},
		),
	}
}

func ApplyWebhookTriggerCleanup(ctx context.Context, cl client.Client, deps *WebhookTriggerDeps, ksr *KnativeServiceResult) error {
	if ksr.Error != nil {
		return errmark.MarkTransient(fmt.Errorf("waiting for Knative service"))
	}

	wtc := NewWebhookTriggerCleanup(deps, ksr.KnativeService)
	if _, err := (lifecycle.RequiredLoader{wtc}).Load(ctx, cl); err != nil {
		return err
	}

	if _, err := wtc.Delete(ctx, cl); err != nil {
		return err
	}

	return nil
}
