package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PersistentVolume struct {
	Key    client.ObjectKey
	Object *corev1.PersistentVolume
}

var _ Persister = &PersistentVolume{}
var _ Loader = &PersistentVolume{}
var _ Ownable = &PersistentVolume{}
var _ LabelAnnotatableFrom = &PersistentVolume{}

func (pv *PersistentVolume) Persist(ctx context.Context, cl client.Client) error {
	if err := Create(ctx, cl, pv.Key, pv.Object); err != nil {
		return err
	}

	_, err := RequiredLoader{pv}.Load(ctx, cl)
	return err
}

func (pv *PersistentVolume) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, pv.Key, pv.Object)
}

func (pv *PersistentVolume) Owned(ctx context.Context, owner Owner) error {
	return Own(pv.Object, owner)
}

func (pv *PersistentVolume) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&pv.Object.ObjectMeta, from)
}

func NewPersistentVolume(key client.ObjectKey) *PersistentVolume {
	return &PersistentVolume{
		Key:    key,
		Object: &corev1.PersistentVolume{},
	}
}

func ApplyPersistentVolume(ctx context.Context, cl client.Client, key client.ObjectKey, pv *corev1.PersistentVolume) (*PersistentVolume, error) {
	p := NewPersistentVolume(key)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if pv != nil {
		p.Object.Spec = pv.Spec
	}

	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}
