package app

import (
	"context"
	"path"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/admission"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AmbassadorIDAnnotation                      = "getambassador.io/ambassador-id"
	ImmutableConfigMapResourceVersionAnnotation = "controller.relay.sh/immutable-config-map-resource-version"

	KnativeServiceVisibilityLabel = "serving.knative.dev/visibility"
)

const (
	AmbassadorID                         = "webhook"
	KnativeServiceVisibilityClusterLocal = "cluster-local"
)

func ConfigureKnativeService(ctx context.Context, s *obj.KnativeService, wtd *WebhookTriggerDeps) error {
	// FIXME This should be configurable
	lifecycle.Annotate(ctx, s, AmbassadorIDAnnotation, AmbassadorID)
	lifecycle.Label(ctx, s, KnativeServiceVisibilityLabel, KnativeServiceVisibilityClusterLocal)
	s.LabelAnnotateFrom(ctx, wtd.WebhookTrigger.Object)

	// Owned by the owner ConfigMap so we only have to worry about deleting one
	// thing.
	if err := wtd.OwnerConfigMap.Own(ctx, s); err != nil {
		return err
	}

	// We also set this as a dependency of the webhook trigger so that changes
	// to the service will propagate back using our event handler.
	if err := DependencyManager.SetDependencyOf(&s.Object.ObjectMeta, lifecycle.TypedObject{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return err
	}

	template := servingv1.RevisionTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				model.RelayControllerWebhookTriggerIDLabel: wtd.WebhookTrigger.Key.Name,
			},
			Annotations: map[string]string{
				// We keep track of the immutable config map version to ensure
				// Knative updates if the `input` script changes.
				//
				// It should be safe to use the resource version here as
				// resource versions aren't supposed to change under semantic
				// equality. However, this has been buggy in previous versions
				// of Kubernetes, so we can always switch to a hash instead if
				// needed.
				ImmutableConfigMapResourceVersionAnnotation: wtd.ImmutableConfigMap.Object.GetResourceVersion(),

				// Keep any existing token annotations.
				model.RelayControllerTokenHashAnnotation: s.Object.Spec.ConfigurationSpec.Template.GetAnnotations()[model.RelayControllerTokenHashAnnotation],
				authenticate.KubernetesTokenAnnotation:   s.Object.Spec.ConfigurationSpec.Template.GetAnnotations()[authenticate.KubernetesTokenAnnotation],
				authenticate.KubernetesSubjectAnnotation: s.Object.Spec.ConfigurationSpec.Template.GetAnnotations()[authenticate.KubernetesSubjectAnnotation],
			},
		},
		Spec: servingv1.RevisionSpec{
			PodSpec: corev1.PodSpec{
				ServiceAccountName: wtd.KnativeServiceAccount.Key.Name,
				EnableServiceLinks: pointer.BoolPtr(false),
			},
		},
	}

	// The revisions will be marked with a dependency reference as well as we
	// need to track them to clean up stale checkouts.
	if err := DependencyManager.SetDependencyOf(&template.ObjectMeta, lifecycle.TypedObject{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return err
	}

	image := wtd.WebhookTrigger.Object.Spec.Image
	if image == "" {
		// Theoretically someone could write some socat action and use the
		// Alpine image, so we leave this here for consistency.
		image = model.DefaultImage
	}

	container := corev1.Container{
		Name:            wtd.WebhookTrigger.Object.Name,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{
			{
				Name:  "CI",
				Value: "true",
			},
			{
				Name:  "RELAY",
				Value: "true",
			},
			{
				Name:  "METADATA_API_URL",
				Value: wtd.MetadataAPIURL.String(),
			},
		},
	}

	command := wtd.WebhookTrigger.Object.Spec.Command
	args := wtd.WebhookTrigger.Object.Spec.Args

	if len(wtd.WebhookTrigger.Object.Spec.Input) > 0 {
		tm := ModelWebhookTrigger(wtd.WebhookTrigger)

		found := false
		config := configVolumeKey(tm)
		for _, volume := range template.Spec.PodSpec.Volumes {
			if volume.Name == config {
				found = true
				break
			}
		}

		if !found {
			template.Spec.PodSpec.Volumes = append(template.Spec.PodSpec.Volumes, corev1.Volume{
				Name: config,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: wtd.ImmutableConfigMap.Key.Name,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  scriptConfigMapKey(tm),
								Path: "input-script",
								Mode: func(i int32) *int32 { return &i }(0755),
							},
						},
					},
				},
			})
		}

		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      config,
				ReadOnly:  true,
				MountPath: "/var/run/puppet/relay/config",
			},
		}

		command = "/var/run/puppet/relay/config/input-script"
		args = []string{}
	}

	if wtd.ToolInjectionCheckout.Satisfied() {
		ep, err := entrypoint.ImageEntrypoint(image, []string{command}, args)
		if err != nil {
			return err
		}

		container.Command = []string{path.Join(model.ToolsMountPath, ep.Entrypoint)}
		container.Args = ep.Args

		helper.Annotate(&template.ObjectMeta, admission.ToolsVolumeClaimAnnotation, wtd.ToolInjectionCheckout.Object.Spec.ClaimName)
	} else {
		if command != "" {
			container.Command = []string{command}
		}

		if len(args) > 0 {
			container.Args = args
		}
	}

	template.Spec.PodSpec.Containers = []corev1.Container{container}

	if err := wtd.AnnotateTriggerToken(ctx, &template.ObjectMeta); err != nil {
		return err
	}

	s.Object.Spec = servingv1.ServiceSpec{
		ConfigurationSpec: servingv1.ConfigurationSpec{
			Template: template,
		},
	}

	return nil
}

func ApplyKnativeService(ctx context.Context, cl client.Client, wtd *WebhookTriggerDeps) (*obj.KnativeService, error) {
	s := obj.NewKnativeService(client.ObjectKey{
		Namespace: wtd.TenantDeps.Namespace.Name,
		Name:      wtd.WebhookTrigger.Key.Name,
	})

	if _, err := s.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigureKnativeService(ctx, s, wtd); err != nil {
		return nil, err
	}

	if err := s.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return s, nil
}

type KnativeServiceResult struct {
	KnativeService *obj.KnativeService
	Error          error
}

func AsKnativeServiceResult(ks *obj.KnativeService, err error) *KnativeServiceResult {
	return &KnativeServiceResult{
		KnativeService: ks,
		Error:          err,
	}
}
