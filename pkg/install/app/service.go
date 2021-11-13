package app

import (
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ConfigureMetadataAPIService(md *MetadataAPIDeps, svc *corev1obj.Service) {
	svc.Object.Spec.Selector = md.Labels

	port := int32(80)
	if md.Core.Object.Spec.MetadataAPI.TLSSecretName != nil {
		port = int32(443)
	}

	svc.Object.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "http",
			Port:       port,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromString("http"),
		},
	}
}
