package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyPersistentVolumeClaim(ctx context.Context, cl client.Client, key client.ObjectKey, pvc *corev1.PersistentVolumeClaim) (*corev1obj.PersistentVolumeClaim, error) {
	p := corev1obj.NewPersistentVolumeClaim(key)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if pvc != nil {
		if helper.Exists(p.Object) {
			p.Object.Spec.Resources = pvc.Spec.Resources
		} else {
			p.Object.Spec = pvc.Spec
		}
	}

	p.LabelAnnotateFrom(ctx, pvc)
	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}

type PersistentVolumeClaimResult struct {
	PersistentVolumeClaim *corev1obj.PersistentVolumeClaim
	Error                 error
}

func AsPersistentVolumeClaimResult(pvc *corev1obj.PersistentVolumeClaim, err error) *PersistentVolumeClaimResult {
	return &PersistentVolumeClaimResult{
		PersistentVolumeClaim: pvc,
		Error:                 err,
	}
}
