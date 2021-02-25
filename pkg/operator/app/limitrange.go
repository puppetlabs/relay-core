package obj

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
