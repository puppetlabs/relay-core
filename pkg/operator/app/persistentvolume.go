package app

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyPersistentVolume(ctx context.Context, cl client.Client, key client.ObjectKey, pv *corev1.PersistentVolume) (*PersistentVolume, error) {
	p := NewPersistentVolume(key)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if pv != nil {
		exists, err := Exists(key, p.Object)
		if err != nil {
			return nil, err
		}

		if exists {
			p.Object.Spec.AccessModes = pv.Spec.AccessModes
			p.Object.Spec.Capacity = pv.Spec.Capacity
			p.Object.Spec.ClaimRef = pv.Spec.ClaimRef
		} else {
			p.Object.Spec = pv.Spec
		}
	}

	p.LabelAnnotateFrom(ctx, pv.ObjectMeta)
	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}

type PersistentVolumeResult struct {
	PersistentVolume *PersistentVolume
	Error            error
}

func AsPersistentVolumeResult(pv *PersistentVolume, err error) *PersistentVolumeResult {
	return &PersistentVolumeResult{
		PersistentVolume: pv,
		Error:            err,
	}
}
