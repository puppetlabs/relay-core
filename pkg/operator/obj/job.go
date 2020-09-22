package obj

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Job struct {
	Key    client.ObjectKey
	Object *batchv1.Job
}

var _ Persister = &Job{}
var _ Loader = &Job{}
var _ Ownable = &Job{}
var _ LabelAnnotatableFrom = &Job{}

func (j *Job) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, j.Key, j.Object)
}

func (j *Job) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, j.Key, j.Object)
}

func (j *Job) Owned(ctx context.Context, owner Owner) error {
	return Own(j.Object, owner)
}

func (j *Job) Delete(ctx context.Context, cl client.Client, opts ...client.DeleteOption) (bool, error) {
	if err := cl.Delete(ctx, j.Object, opts...); k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (j *Job) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&j.Object.ObjectMeta, from)
}

func NewJob(key client.ObjectKey) *Job {
	return &Job{
		Key:    key,
		Object: &batchv1.Job{},
	}
}

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
