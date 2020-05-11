package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LimitRange struct {
	Key    client.ObjectKey
	Object *corev1.LimitRange
}

var _ Persister = &LimitRange{}
var _ Loader = &LimitRange{}
var _ Ownable = &LimitRange{}

func (lr *LimitRange) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, lr.Key, lr.Object)
}

func (lr *LimitRange) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, lr.Key, lr.Object)
}

func (lr *LimitRange) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&lr.Object.ObjectMeta, ref)
}

func NewLimitRange(key client.ObjectKey) *LimitRange {
	return &LimitRange{
		Key:    key,
		Object: &corev1.LimitRange{},
	}
}

type limitRangeOptions struct {
	containerDefaultLimit        corev1.ResourceList
	containerDefaultRequestLimit corev1.ResourceList
	containerMaxLimit            corev1.ResourceList
}

type LimitRangeOption func(opts *limitRangeOptions)

func LimitRangeWithContainerDefaultLimit(rl corev1.ResourceList) LimitRangeOption {
	return func(opts *limitRangeOptions) {
		opts.containerDefaultLimit = rl
	}
}

func LimitRangeWithContainerDefaultRequestLimit(rl corev1.ResourceList) LimitRangeOption {
	return func(opts *limitRangeOptions) {
		opts.containerDefaultRequestLimit = rl
	}
}

func LimitRangeWithContainerMaxLimit(rl corev1.ResourceList) LimitRangeOption {
	return func(opts *limitRangeOptions) {
		opts.containerMaxLimit = rl
	}
}

func ConfigureLimitRange(lr *LimitRange, opts ...LimitRangeOption) {
	lro := &limitRangeOptions{
		containerDefaultLimit: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("750m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		containerDefaultRequestLimit: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		containerMaxLimit: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("3Gi"),
		},
	}

	for _, opt := range opts {
		opt(lro)
	}

	lr.Object.Spec = corev1.LimitRangeSpec{
		Limits: []corev1.LimitRangeItem{
			{
				Type:           corev1.LimitTypeContainer,
				Default:        lro.containerDefaultLimit,
				DefaultRequest: lro.containerDefaultRequestLimit,
				Max:            lro.containerMaxLimit,
			},
		},
	}
}
