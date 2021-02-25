package obj

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyJob(ctx context.Context, cl client.Client, key client.ObjectKey, job *batchv1.Job) (*Job, error) {
	j := NewJob(key)

	if _, err := j.Load(ctx, cl); err != nil {
		return nil, err
	}

	if job != nil {
		exists, err := Exists(key, j.Object)
		if err != nil {
			return nil, err
		}

		if exists {
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
