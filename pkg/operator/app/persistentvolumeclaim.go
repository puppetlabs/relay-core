package app

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
