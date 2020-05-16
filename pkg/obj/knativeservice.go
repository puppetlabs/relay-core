package obj

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AmbassadorIdAnnotation        = "getambassador.io/ambassador-id"
	KnativeServiceVisibilityLabel = "serving.knative.dev/visibility"
)

const (
	AmbassadorId                         = "webhook"
	KnativeServiceVisibilityClusterLocal = "cluster-local"
)

var (
	KnativeServiceKind = servingv1.SchemeGroupVersion.WithKind("Service")
)

type KnativeService struct {
	Key    client.ObjectKey
	Object *servingv1.Service
}

var _ Persister = &KnativeService{}
var _ Loader = &KnativeService{}
var _ Ownable = &KnativeService{}
var _ LabelAnnotatableFrom = &KnativeService{}

func (ks *KnativeService) Persist(ctx context.Context, cl client.Client) error {
	if err := CreateOrUpdate(ctx, cl, ks.Key, ks.Object); err != nil {
		return err
	}

	return nil
}

func (ks *KnativeService) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, ks.Key, ks.Object)
}

func (ks *KnativeService) Owned(ctx context.Context, owner Owner) error {
	return Own(ks.Object, owner)
}

func (ks *KnativeService) Label(ctx context.Context, name, value string) {
	Label(&ks.Object.ObjectMeta, name, value)
}

func (ks *KnativeService) Annotate(ctx context.Context, name, value string) {
	Annotate(&ks.Object.ObjectMeta, name, value)
}

func (ks *KnativeService) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&ks.Object.ObjectMeta, from)
}

func NewKnativeService(key client.ObjectKey) *KnativeService {
	return &KnativeService{
		Key:    key,
		Object: &servingv1.Service{},
	}
}

func ConfigureKnativeService(ctx context.Context, s *KnativeService, wtd *WebhookTriggerDeps) error {
	// FIXME This should be configurable
	s.Annotate(ctx, AmbassadorIdAnnotation, AmbassadorId)
	s.Label(ctx, KnativeServiceVisibilityLabel, KnativeServiceVisibilityClusterLocal)
	s.LabelAnnotateFrom(ctx, wtd.WebhookTrigger.Object.ObjectMeta)

	if err := wtd.WebhookTrigger.Own(ctx, s); err != nil {
		return err
	}

	container := corev1.Container{
		Name:            wtd.WebhookTrigger.Object.Name,
		Image:           wtd.WebhookTrigger.Object.Spec.Image,
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{
			{
				Name:  "METADATA_API_URL",
				Value: wtd.MetadataAPIURL.String(),
			},
		},
	}

	if command := wtd.WebhookTrigger.Object.Spec.Command; command != "" {
		container.Command = []string{command}
	}

	if args := wtd.WebhookTrigger.Object.Spec.Args; len(args) > 0 {
		container.Args = args
	}

	template := servingv1.RevisionTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				model.RelayControllerWebhookTriggerIDLabel: wtd.WebhookTrigger.Key.Name,
			},
			// Keep any existing token annotations.
			Annotations: map[string]string{
				authenticate.KubernetesTokenAnnotation:   s.Object.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[authenticate.KubernetesTokenAnnotation],
				authenticate.KubernetesSubjectAnnotation: s.Object.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[authenticate.KubernetesSubjectAnnotation],
			},
		},
		Spec: servingv1.RevisionSpec{
			PodSpec: corev1.PodSpec{
				ServiceAccountName: wtd.KnativeServiceAccount.Key.Name,
				Containers:         []corev1.Container{container},
			},
		},
	}

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

func ApplyKnativeService(ctx context.Context, cl client.Client, wtd *WebhookTriggerDeps) (*KnativeService, error) {
	s := NewKnativeService(wtd.WebhookTrigger.Key)

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
	KnativeService *KnativeService
	Error          error
}

func AsKnativeServiceResult(ks *KnativeService, err error) *KnativeServiceResult {
	return &KnativeServiceResult{
		KnativeService: ks,
		Error:          err,
	}
}
