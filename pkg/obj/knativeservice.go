package obj

import (
	"context"

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

func (ks *KnativeService) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&ks.Object.ObjectMeta, ref)
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

func ApplyKnativeService(ctx context.Context, cl client.Client, wt *WebhookTrigger, wtd *WebhookTriggerDeps) (*KnativeService, error) {
	s := NewKnativeService(client.ObjectKey{Name: wt.Object.Name, Namespace: wt.Object.Namespace})

	if _, err := s.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigureKnativeService(ctx, s, wt, wtd); err != nil {
		return nil, err
	}

	if err := s.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return s, nil
}

func ConfigureKnativeService(ctx context.Context, s *KnativeService, wt *WebhookTrigger, wtd *WebhookTriggerDeps) error {
	// FIXME This should be configurable
	s.Annotate(ctx, AmbassadorIdAnnotation, AmbassadorId)
	s.Label(ctx, KnativeServiceVisibilityLabel, KnativeServiceVisibilityClusterLocal)
	s.LabelAnnotateFrom(ctx, wt.Object.ObjectMeta)

	wt.Own(ctx, s)

	s.Object.Spec = servingv1.ServiceSpec{
		ConfigurationSpec: servingv1.ConfigurationSpec{
			Template: servingv1.RevisionTemplateSpec{
				Spec: servingv1.RevisionSpec{
					PodSpec: corev1.PodSpec{
						ServiceAccountName: wtd.SystemServiceAccount.Key.Name,
						Containers: []corev1.Container{
							{
								Name:            wt.Object.Name,
								Image:           wt.Object.Spec.Image,
								ImagePullPolicy: corev1.PullAlways,
								Env: []corev1.EnvVar{
									{
										Name:  "METADATA_API_URL",
										Value: wtd.MetadataAPIURL.String(),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := wtd.AnnotateTriggerToken(ctx, &s.Object.ObjectMeta); err != nil {
		return err
	}

	return nil
}
