package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ServingKind = servingv1.SchemeGroupVersion.WithKind("Serving")
)

type Serving struct {
	Key    client.ObjectKey
	Object *servingv1.Service
}

var _ Persister = &Serving{}
var _ Loader = &Serving{}
var _ Ownable = &Serving{}
var _ LabelAnnotatableFrom = &Serving{}

func (s *Serving) Persist(ctx context.Context, cl client.Client) error {
	if err := CreateOrUpdate(ctx, cl, s.Key, s.Object); err != nil {
		return err
	}

	if err := cl.Status().Update(ctx, s.Object); err != nil {
		return err
	}

	return nil
}

func (s *Serving) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, s.Key, s.Object)
}

func (n *Serving) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&n.Object.ObjectMeta, ref)
}

func (n *Serving) Label(ctx context.Context, name, value string) {
	Label(&n.Object.ObjectMeta, name, value)
}

func (n *Serving) Annotate(ctx context.Context, name, value string) {
	Annotate(&n.Object.ObjectMeta, name, value)
}

func (n *Serving) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&n.Object.ObjectMeta, from)
}

func NewServing(key client.ObjectKey) *Serving {
	return &Serving{
		Key:    key,
		Object: &servingv1.Service{},
	}
}

func ApplyServing(ctx context.Context, cl client.Client, wt *WebhookTrigger, wtd *WebhookTriggerDeps) (*Serving, error) {
	s := NewServing(client.ObjectKey{Name: wt.Object.Name, Namespace: wt.Object.Namespace})

	if _, err := s.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigureServing(ctx, s, wt, wtd); err != nil {
		return nil, err
	}

	if err := s.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return s, nil
}

func ConfigureServing(ctx context.Context, s *Serving, wt *WebhookTrigger, wtd *WebhookTriggerDeps) error {
	// FIXME This should be configurable
	s.Annotate(ctx, "getambassador.io/ambassador-id", "webhook")
	s.Label(ctx, "serving.knative.dev/visibility", "cluster-local")
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
