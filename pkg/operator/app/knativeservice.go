package app

import (
	"context"
	"path"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func ConfigureKnativeService(ctx context.Context, s *KnativeService, wtd *WebhookTriggerDeps) error {
	// FIXME This should be configurable
	s.Annotate(ctx, AmbassadorIDAnnotation, AmbassadorID)
	s.Label(ctx, KnativeServiceVisibilityLabel, KnativeServiceVisibilityClusterLocal)
	s.LabelAnnotateFrom(ctx, wtd.WebhookTrigger.Object.ObjectMeta)

	// Owned by the owner ConfigMap so we only have to worry about deleting one
	// thing.
	if err := wtd.OwnerConfigMap.Own(ctx, s); err != nil {
		return err
	}

	// We also set this as a dependency of the webhook trigger so that changes
	// to the service will propagate back using our event handler.
	SetDependencyOf(&s.Object.ObjectMeta, Owner{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind})

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
			},
		},
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

	if wtd.Tenant.Object.Spec.ToolInjection.VolumeClaimTemplate != nil {
		ep, err := entrypoint.ImageEntrypoint(image, []string{command}, args)
		if err != nil {
			return err
		}

		container.Command = []string{path.Join(model.ToolInjectionMountPath, ep.Entrypoint)}
		container.Args = ep.Args
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

	if wtd.Tenant.Object.Spec.ToolInjection.VolumeClaimTemplate != nil {
		Annotate(&template.ObjectMeta, model.RelayControllerToolsVolumeClaimAnnotation, wtd.Tenant.Object.GetName()+model.ToolInjectionVolumeClaimSuffixReadOnlyMany)
	}

	s.Object.Spec = servingv1.ServiceSpec{
		ConfigurationSpec: servingv1.ConfigurationSpec{
			Template: template,
		},
	}

	return nil
}

func ApplyKnativeService(ctx context.Context, cl client.Client, wtd *WebhookTriggerDeps) (*KnativeService, error) {
	s := NewKnativeService(client.ObjectKey{
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
