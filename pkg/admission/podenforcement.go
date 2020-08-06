package admission

import (
	"context"
	"encoding/json"
	"net/http"

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
	PodSandboxRuntimeClassName = "gvisor"
)

type PodEnforcementHandler struct {
	sandboxed  bool
	standalone bool
	decoder    *admission.Decoder
}

var _ admission.Handler = &PodEnforcementHandler{}
var _ admission.DecoderInjector = &PodEnforcementHandler{}

func (peh *PodEnforcementHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := peh.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	pod.Spec.NodeSelector = PodNodeSelector
	pod.Spec.Tolerations = PodTolerations

	if !peh.standalone {
		pod.Spec.DNSPolicy = PodDNSPolicy
		pod.Spec.DNSConfig = PodDNSConfig
	}

	if peh.sandboxed {
		pod.Spec.RuntimeClassName = &PodSandboxRuntimeClassName
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

func PodEnforcementHandlerWithSandboxing(sandboxed bool) PodEnforcementHandlerOption {
	return func(peh *PodEnforcementHandler) {
		peh.sandboxed = sandboxed
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
