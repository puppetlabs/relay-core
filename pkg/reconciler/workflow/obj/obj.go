package obj

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrRequired = errors.New("obj: required")
)

type Persister interface {
	Persist(ctx context.Context, cl client.Client) error
}

type IgnoreNilPersister struct {
	Persister
}

func (inp IgnoreNilPersister) Persist(ctx context.Context, cl client.Client) error {
	if inp.Persister == nil {
		return nil
	}

	return inp.Persister.Persist(ctx, cl)
}

type Loader interface {
	Load(ctx context.Context, cl client.Client) (bool, error)
}

type IgnoreNilLoader struct {
	Loader
}

func (inl IgnoreNilLoader) Load(ctx context.Context, cl client.Client) (bool, error) {
	if inl.Loader == nil {
		return true, nil
	}

	return inl.Loader.Load(ctx, cl)
}

type RequiredLoader struct {
	Loader
}

func (rl RequiredLoader) Load(ctx context.Context, cl client.Client) (bool, error) {
	ok, err := rl.Loader.Load(ctx, cl)
	if err != nil {
		return false, err
	} else if !ok {
		return false, ErrRequired
	}

	return true, nil
}

type Loaders []Loader

var _ Loader = Loaders(nil)

func (ls Loaders) Load(ctx context.Context, cl client.Client) (bool, error) {
	all := true

	for _, l := range ls {
		if ok, err := l.Load(ctx, cl); err != nil {
			return false, err
		} else if !ok {
			all = false
		}
	}

	return all, nil
}

type Ownable interface {
	Owned(ctx context.Context, ref *metav1.OwnerReference)
}

type IgnoreNilOwnable struct {
	Ownable
}

func (ino IgnoreNilOwnable) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	if ino.Ownable == nil {
		return
	}

	ino.Ownable.Owned(ctx, ref)
}

type LabelAnnotatableFrom interface {
	LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta)
}

type IgnoreNilLabelAnnotatableFrom struct {
	LabelAnnotatableFrom
}

func (inlaf IgnoreNilLabelAnnotatableFrom) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	if inlaf.LabelAnnotatableFrom == nil {
		return
	}

	inlaf.LabelAnnotateFrom(ctx, from)
}
