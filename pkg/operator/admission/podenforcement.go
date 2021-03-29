package admission

import (
	"context"
	"encoding/json"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	PodNodeSelector = map[string]string{
		"nebula.puppet.com/scheduling.customer-ready": "true",
	}
	PodTolerations = []corev1.Toleration{
		{
			Key:    "nebula.puppet.com/scheduling.customer-workload",
			Value:  "true",
			Effect: corev1.TaintEffectNoSchedule,
		},
		{
			Key:    "sandbox.gke.io/runtime",
			Value:  "gvisor",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}
	PodDNSPolicy = corev1.DNSNone
	PodDNSConfig = &corev1.PodDNSConfig{
		Nameservers: []string{
			"1.1.1.1",
			"1.0.0.1",
			"8.8.8.8",
		},
	}
)

type PodEnforcementHandler struct {
	runtimeClassName string
	standalone       bool
	decoder          *admission.Decoder
}

var _ admission.Handler = &PodEnforcementHandler{}
var _ admission.DecoderInjector = &PodEnforcementHandler{}

func (peh *PodEnforcementHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("")
	}

	pod := &corev1.Pod{}
	if err := peh.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if !peh.standalone {
		if pod.Spec.NodeSelector == nil {
			pod.Spec.NodeSelector = map[string]string{}
		}
		for key, value := range PodNodeSelector {
			pod.Spec.NodeSelector[key] = value
		}
		pod.Spec.Tolerations = PodTolerations

		pod.Spec.DNSPolicy = PodDNSPolicy
		pod.Spec.DNSConfig = PodDNSConfig
	}

	if peh.runtimeClassName != "" {
		pod.Spec.RuntimeClassName = &peh.runtimeClassName
	}

	b, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, b)
}

func (peh *PodEnforcementHandler) InjectDecoder(d *admission.Decoder) error {
	peh.decoder = d
	return nil
}

type PodEnforcementHandlerOption func(peh *PodEnforcementHandler)

func PodEnforcementHandlerWithRuntimeClassName(runtimeClassName string) PodEnforcementHandlerOption {
	return func(peh *PodEnforcementHandler) {
		peh.runtimeClassName = runtimeClassName
	}
}

func PodEnforcementHandlerWithStandaloneMode(standalone bool) PodEnforcementHandlerOption {
	return func(peh *PodEnforcementHandler) {
		peh.standalone = standalone
	}
}

func NewPodEnforcementHandler(opts ...PodEnforcementHandlerOption) *PodEnforcementHandler {
	peh := &PodEnforcementHandler{}

	for _, opt := range opts {
		opt(peh)
	}

	return peh
}
