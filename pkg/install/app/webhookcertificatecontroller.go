package app

import (
	"strconv"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConfigureWebhookCertificateControllerDeployment(wd *WebhookCertificateControllerDeps, dep *appsv1obj.Deployment) {
	core := wd.Core.Object

	if dep.Object.Labels == nil {
		dep.Object.Labels = make(map[string]string)
	}

	for k, v := range wd.Labels {
		dep.Object.Labels[k] = v
	}

	dep.Object.Spec.Template.Labels = wd.Labels
	dep.Object.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: wd.Labels,
	}

	template := &dep.Object.Spec.Template.Spec
	template.ServiceAccountName = dep.Key.Name

	// This currently uses the same node selector as the operator deployment.
	// This may change in the future if there's a need (such as a separate node
	// pool for secure cert generation).
	template.NodeSelector = core.Spec.Operator.NodeSelector

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 1)
	}

	sc := corev1.Container{}

	ConfigureWebhookCertificateControllerContainer(wd, &sc)

	template.Containers[0] = sc
}

func ConfigureWebhookCertificateControllerContainer(wd *WebhookCertificateControllerDeps, c *corev1.Container) {
	core := wd.Core.Object

	c.Name = "operator-webhook-certificate-controller"
	c.Image = core.Spec.Operator.AdmissionWebhookServer.CertificateControllerImage
	c.ImagePullPolicy = core.Spec.Operator.AdmissionWebhookServer.CertificateControllerImagePullPolicy

	env := []corev1.EnvVar{
		{
			Name:  "RELAY_OPERATOR_DEBUG",
			Value: strconv.FormatBool(core.Spec.Debug),
		},
		{
			Name:  "RELAY_OPERATOR_NAME",
			Value: wd.TargetDeployment.Name,
		},
		{
			Name:  "RELAY_OPERATOR_NAMESPACE",
			Value: wd.TargetDeployment.Namespace,
		},
		{
			Name:  "RELAY_OPERATOR_SERVICE_NAME",
			Value: wd.TargetDeployment.Name,
		},
		{
			Name:  "RELAY_OPERATOR_CERTIFICATE_SECRET_NAME",
			Value: helper.SuffixObjectKey(wd.TargetDeployment, "certificate").Name,
		},
		{
			Name:  "RELAY_OPERATOR_MUTATING_WEBHOOK_CONFIGURATION_NAME",
			Value: wd.TargetDeployment.Name,
		},
	}

	c.Env = env
}
