package app

import (
	"fmt"

	admissionregistrationv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/admissionregistrationv1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
)

func ConfigureOperatorWebhookConfiguration(od *OperatorDeps, mwc *admissionregistrationv1obj.MutatingWebhookConfiguration) {
	var (
		podEnforcementPath = "/mutate/pod-enforcement"
		volumeClaimPath    = "/mutate/volume-claim"
		failurePolicy      = admissionv1.Fail
		sideEffects        = admissionv1.SideEffectClassNone
		reinvocationPolicy = admissionv1.IfNeededReinvocationPolicy
	)

	oc := od.Core.Object.Spec.Operator
	aws := oc.AdmissionWebhookServer

	mwc.Object.Webhooks = []admissionv1.MutatingWebhook{
		{
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			Name:                    fmt.Sprintf("%s-pod-enforcement.%s", mwc.Name, aws.Domain),
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      od.WebhookService.Key.Name,
					Namespace: od.WebhookService.Key.Namespace,
					Path:      &podEnforcementPath,
				},
			},
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{
						admissionv1.Create, admissionv1.Update,
					},
					Rule: admissionv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			FailurePolicy:      &failurePolicy,
			SideEffects:        &sideEffects,
			ReinvocationPolicy: &reinvocationPolicy,
			NamespaceSelector:  oc.AdmissionWebhookServer.NamespaceSelector,
		},
		{
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
			Name:                    fmt.Sprintf("%s-volume-claim.%s", mwc.Name, aws.Domain),
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      od.WebhookService.Key.Name,
					Namespace: od.WebhookService.Key.Namespace,
					Path:      &volumeClaimPath,
				},
			},
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{admissionv1.Create, admissionv1.Update},
					Rule: admissionv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				},
			},
			FailurePolicy:      &failurePolicy,
			SideEffects:        &sideEffects,
			ReinvocationPolicy: &reinvocationPolicy,
			NamespaceSelector:  oc.AdmissionWebhookServer.NamespaceSelector,
		},
	}
}
