package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PersistentVolumeClaim struct {
	Key    client.ObjectKey
	Object *corev1.PersistentVolumeClaim
}

var _ Persister = &PersistentVolumeClaim{}
var _ Loader = &PersistentVolumeClaim{}
var _ Deleter = &PersistentVolumeClaim{}
var _ Ownable = &PersistentVolumeClaim{}
var _ LabelAnnotatableFrom = &PersistentVolumeClaim{}

func (pvc *PersistentVolumeClaim) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, pvc.Key, pvc.Object)
}

func (pvc *PersistentVolumeClaim) Patch(ctx context.Context, cl client.Client, original *corev1.PersistentVolumeClaim) error {
	return Patch(ctx, cl, pvc.Key, pvc.Object, original)
}

func (pvc *PersistentVolumeClaim) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, pvc.Key, pvc.Object)
}

func (pvc *PersistentVolumeClaim) Delete(ctx context.Context, cl client.Client) (bool, error) {
	return DeleteIgnoreNotFound(ctx, cl, pvc.Object)
}

func (pvc *PersistentVolumeClaim) Owned(ctx context.Context, owner Owner) error {
	return Own(pvc.Object, owner)
}

func (pvc *PersistentVolumeClaim) Label(ctx context.Context, name, value string) {
	Label(&pvc.Object.ObjectMeta, name, value)
}

func (pvc *PersistentVolumeClaim) Annotate(ctx context.Context, name, value string) {
	Annotate(&pvc.Object.ObjectMeta, name, value)
}

func (pvc *PersistentVolumeClaim) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&pvc.Object.ObjectMeta, from)
}

func NewPersistentVolumeClaim(key client.ObjectKey) *PersistentVolumeClaim {
	return &PersistentVolumeClaim{
		Key:    key,
		Object: &corev1.PersistentVolumeClaim{},
	}
}

func ApplyPersistentVolumeClaim(ctx context.Context, cl client.Client, key client.ObjectKey, pvc *corev1.PersistentVolumeClaim) (*PersistentVolumeClaim, error) {
	p := NewPersistentVolumeClaim(key)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if pvc != nil {
		exists, err := Exists(key, p.Object)
		if err != nil {
			return nil, err
		}

		if exists {
			p.Object.Spec.Resources = pvc.Spec.Resources
		} else {
			p.Object.Spec = pvc.Spec
		}
	}

	p.LabelAnnotateFrom(ctx, pvc.ObjectMeta)
	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}

type PersistentVolumeClaimResult struct {
	PersistentVolumeClaim *PersistentVolumeClaim
	Error                 error
}

func AsPersistentVolumeClaimResult(pvc *PersistentVolumeClaim, err error) *PersistentVolumeClaimResult {
	return &PersistentVolumeClaimResult{
		PersistentVolumeClaim: pvc,
		Error:                 err,
	}
}
