package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyPersistentVolume(ctx context.Context, cl client.Client, name string, pv *corev1.PersistentVolume) (*corev1obj.PersistentVolume, error) {
	p := corev1obj.NewPersistentVolume(name)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if pv != nil {
		if helper.Exists(p.Object) {
			p.Object.Spec.AccessModes = pv.Spec.AccessModes
			p.Object.Spec.Capacity = pv.Spec.Capacity
			p.Object.Spec.ClaimRef = pv.Spec.ClaimRef
		} else {
			p.Object.Spec = pv.Spec
		}
	}

	p.LabelAnnotateFrom(ctx, pv)
	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}

type PersistentVolumeResult struct {
	PersistentVolume *corev1obj.PersistentVolume
	Error            error
}

func AsPersistentVolumeResult(pv *corev1obj.PersistentVolume, err error) *PersistentVolumeResult {
	return &PersistentVolumeResult{
		PersistentVolume: pv,
		Error:            err,
	}
}
