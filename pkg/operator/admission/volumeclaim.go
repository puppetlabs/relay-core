package admission

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/puppetlabs/relay-core/pkg/model"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ToolsVolumeClaimAnnotation = "controller.relay.sh/tools-volume-claim"
	ToolsMountName             = "relay-tools"
)

type VolumeClaimHandler struct {
	decoder *admission.Decoder
}

var _ admission.Handler = &VolumeClaimHandler{}
var _ admission.DecoderInjector = &VolumeClaimHandler{}

func (eh *VolumeClaimHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("")
	}

	pod := &corev1.Pod{}
	if err := eh.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if claim, ok := pod.ObjectMeta.GetAnnotations()[ToolsVolumeClaimAnnotation]; ok {
		cs := make([]corev1.Container, 0)

		updated := false
		for _, c := range pod.Spec.Containers {
			hasVolumeMount := false
			for _, vm := range c.VolumeMounts {
				if vm.Name == ToolsMountName {
					hasVolumeMount = true
					break
				}
			}

			if !hasVolumeMount {
				c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
					Name:      ToolsMountName,
					MountPath: model.ToolsMountPath,
					ReadOnly:  true,
				})

				updated = true
			}

			cs = append(cs, c)
		}

		if updated {
			pod.Spec.Containers = cs
		}

		hasVolume := false
		for _, volume := range pod.Spec.Volumes {
			if volume.Name == ToolsMountName {
				hasVolume = true
				break
			}
		}

		if !hasVolume {
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
				Name: ToolsMountName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: claim,
						ReadOnly:  true,
					},
				},
			})

			updated = true
		}

		if updated {
			b, err := json.Marshal(pod)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}

			return admission.PatchResponseFromRaw(req.Object.Raw, b)
		}
	}

	return admission.Allowed("")
}

func (eh *VolumeClaimHandler) InjectDecoder(d *admission.Decoder) error {
	eh.decoder = d
	return nil
}

type VolumeClaimHandlerOption func(eh *VolumeClaimHandler)

func NewVolumeClaimHandler(opts ...VolumeClaimHandlerOption) *VolumeClaimHandler {
	eh := &VolumeClaimHandler{}

	for _, opt := range opts {
		opt(eh)
	}

	return eh
}
