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
	)

	oc := od.Core.Object.Spec.Operator
	aws := oc.AdmissionWebhookServer

	mutatingWebhooks := make(map[string]*admissionv1.MutatingWebhook)
	for _, mw := range mwc.Object.Webhooks {
		mutatingWebhooks[mw.Name] = mw.DeepCopy()
	}

	podEnforcementName := fmt.Sprintf("%s-pod-enforcement.%s", mwc.Name, aws.Domain)
	volumeClaimName := fmt.Sprintf("%s-volume-claim.%s", mwc.Name, aws.Domain)
	paths := map[string]*string{
		podEnforcementName: &podEnforcementPath,
		volumeClaimName:    &volumeClaimPath,
	}

	for name, path := range paths {
		mutatingWebhook, ok := mutatingWebhooks[name]
		if !ok {
			mutatingWebhook = &admissionv1.MutatingWebhook{}
		}

		ConfigureMutatingWebhook(od, mutatingWebhook, name, path)

		mutatingWebhooks[name] = mutatingWebhook
	}

	mwc.Object.Webhooks = []admissionv1.MutatingWebhook{
		*mutatingWebhooks[podEnforcementName],
		*mutatingWebhooks[volumeClaimName],
	}
}

func ConfigureMutatingWebhook(od *OperatorDeps, mw *admissionv1.MutatingWebhook, name string, path *string) {
	var (
		failurePolicy      = admissionv1.Fail
		sideEffects        = admissionv1.SideEffectClassNone
		reinvocationPolicy = admissionv1.IfNeededReinvocationPolicy
	)

	oc := od.Core.Object.Spec.Operator

	mw.AdmissionReviewVersions = []string{"v1", "v1beta1"}
	mw.Name = name

	mw.ClientConfig.Service = &admissionv1.ServiceReference{
		Name:      od.WebhookService.Key.Name,
		Namespace: od.WebhookService.Key.Namespace,
		Path:      path,
	}

	mw.Rules = []admissionv1.RuleWithOperations{
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
	}

	mw.FailurePolicy = &failurePolicy
	mw.SideEffects = &sideEffects
	mw.ReinvocationPolicy = &reinvocationPolicy
	mw.NamespaceSelector = oc.AdmissionWebhookServer.NamespaceSelector
}
