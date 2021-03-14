package app

import (
	"fmt"

	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureTenant(t *obj.Tenant, td *TenantDepsResult) {
	// Set up our initial map from the existing data.
	conds := map[relayv1beta1.TenantConditionType]*relayv1beta1.Condition{
		relayv1beta1.TenantNamespaceReady:     &relayv1beta1.Condition{},
		relayv1beta1.TenantEventSinkReady:     &relayv1beta1.Condition{},
		relayv1beta1.TenantToolInjectionReady: &relayv1beta1.Condition{},
		relayv1beta1.TenantReady:              &relayv1beta1.Condition{},
	}

	for _, cond := range t.Object.Status.Conditions {
		*conds[cond.Type] = cond.Condition
	}

	// Update with dependency data.
	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.TenantNamespaceReady], func() relayv1beta1.Condition {
		if td.Error != nil {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  obj.TenantStatusReasonNamespaceError,
				Message: td.Error.Error(),
			}
		} else if td.TenantDeps != nil {
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.TenantStatusReasonNamespaceReady,
				Message: "The tenant namespace is ready.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.TenantEventSinkReady], func() relayv1beta1.Condition {
		if td.TenantDeps != nil {
			if sink := td.TenantDeps.APITriggerEventSink; sink != nil {
				if sink.URL() == "" {
					return relayv1beta1.Condition{
						Status:  corev1.ConditionFalse,
						Reason:  obj.TenantStatusReasonEventSinkNotConfigured,
						Message: "The API trigger event sink is missing an endpoint URL.",
					}
				} else if _, ok := sink.Token(); !ok {
					return relayv1beta1.Condition{
						Status:  corev1.ConditionFalse,
						Reason:  obj.TenantStatusReasonEventSinkNotConfigured,
						Message: "The API trigger event sink is missing a token.",
					}
				}

				return relayv1beta1.Condition{
					Status:  corev1.ConditionTrue,
					Reason:  obj.TenantStatusReasonEventSinkReady,
					Message: "The event sink is ready.",
				}
			}

			// This shouldn't block people who want to use these APIs without
			// WebhookTriggers.
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.TenantStatusReasonEventSinkMissing,
				Message: "The tenant does not have an event sink defined.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.TenantToolInjectionReady], func() relayv1beta1.Condition {
		if td.TenantDeps != nil {
			if co := td.TenantDeps.ToolInjectionCheckout; co != nil {
				for _, cond := range co.Object.Status.Conditions {
					if cond.Type != pvpoolv1alpha1.CheckoutAcquired {
						continue
					}

					switch cond.Status {
					case corev1.ConditionTrue:
						return relayv1beta1.Condition{
							Status:  corev1.ConditionTrue,
							Reason:  obj.TenantStatusReasonToolInjectionReady,
							Message: cond.Message,
						}
					case corev1.ConditionFalse:
						return relayv1beta1.Condition{
							Status:  corev1.ConditionFalse,
							Reason:  obj.TenantStatusReasonToolInjectionError,
							Message: fmt.Sprintf("%s: %s", cond.Reason, cond.Message),
						}
					}
					break
				}

				return relayv1beta1.Condition{
					Status: corev1.ConditionUnknown,
				}
			}

			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.TenantStatusReasonToolInjectionNotDefined,
				Message: "The tenant tool injection in not defined.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	UpdateStatusConditionIfTransitioned(conds[relayv1beta1.TenantReady], func() relayv1beta1.Condition {
		switch AggregateStatusConditions(*conds[relayv1beta1.TenantNamespaceReady], *conds[relayv1beta1.TenantEventSinkReady], *conds[relayv1beta1.TenantToolInjectionReady]) {
		case corev1.ConditionTrue:
			return relayv1beta1.Condition{
				Status:  corev1.ConditionTrue,
				Reason:  obj.TenantStatusReasonReady,
				Message: "The tenant is configured.",
			}
		case corev1.ConditionFalse:
			return relayv1beta1.Condition{
				Status:  corev1.ConditionFalse,
				Reason:  obj.TenantStatusReasonError,
				Message: "One or more tenant components failed.",
			}
		}

		return relayv1beta1.Condition{
			Status: corev1.ConditionUnknown,
		}
	})

	t.Object.Status = relayv1beta1.TenantStatus{
		Conditions: []relayv1beta1.TenantCondition{
			{
				Condition: *conds[relayv1beta1.TenantNamespaceReady],
				Type:      relayv1beta1.TenantNamespaceReady,
			},
			{
				Condition: *conds[relayv1beta1.TenantEventSinkReady],
				Type:      relayv1beta1.TenantEventSinkReady,
			},
			{
				Condition: *conds[relayv1beta1.TenantToolInjectionReady],
				Type:      relayv1beta1.TenantToolInjectionReady,
			},
			{
				Condition: *conds[relayv1beta1.TenantReady],
				Type:      relayv1beta1.TenantReady,
			},
		},
	}

	if td.TenantDeps != nil {
		t.Object.Status.ObservedGeneration = t.Object.GetGeneration()
		t.Object.Status.Namespace = td.TenantDeps.Namespace.Name

		if co := td.TenantDeps.ToolInjectionCheckout; co != nil {
			t.Object.Status.ToolInjection.Checkout = corev1.LocalObjectReference{
				Name: co.Key.Name,
			}
		} else {
			t.Object.Status.ToolInjection.Checkout = corev1.LocalObjectReference{}
		}
	}
}
