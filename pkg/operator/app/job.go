package app

import (
	"context"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyJob(ctx context.Context, cl client.Client, key client.ObjectKey, job *batchv1.Job) (*batchv1obj.Job, error) {
	j := batchv1obj.NewJob(key)

	if _, err := j.Load(ctx, cl); err != nil {
		return nil, err
	}

	if job != nil {
		if helper.Exists(j.Object) {
			j.Object.Spec.Template.Spec = job.Spec.Template.Spec
		} else {
			j.Object.Spec = job.Spec
		}
	}

	if err := j.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return j, nil
}
