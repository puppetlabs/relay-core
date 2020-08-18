package obj

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
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
	if err := Create(ctx, cl, j.Key, j.Object); err != nil {
		return err
	}

	_, err := RequiredLoader{j}.Load(ctx, cl)
	return err
}

func (j *Job) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, j.Key, j.Object)
}

func (j *Job) Owned(ctx context.Context, owner Owner) error {
	return Own(j.Object, owner)
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

func ApplyJob(ctx context.Context, cl client.Client, key client.ObjectKey, j *batchv1.Job) (*Job, error) {
	p := NewJob(key)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	if j != nil {
		p.Object.Spec = j.Spec
	}

	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}
