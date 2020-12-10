package controller

import (
	"fmt"
	"net/url"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type component string

func (c component) String() string {
	return string(c)
}

const (
	componentLogService  component = "log-service"
	componentOperator    component = "operator"
	componentMetadataAPI component = "metadata-api"
)

func setDeploymentLabels(labels map[string]string, deployment *appsv1.Deployment) {
	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}

	for k, v := range labels {
		deployment.Labels[k] = v
	}

	deployment.Spec.Template.Labels = labels
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}
}

func metadataAPIURL(rc *installerv1alpha1.RelayCore) string {
	if rc.Spec.MetadataAPI.URL != nil {
		return *rc.Spec.MetadataAPI.URL
	}

	scheme := "http"
	if rc.Spec.MetadataAPI.TLSSecretName != nil {
		scheme = "https"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s-metadata-api.%s.svc.cluster.local", rc.Name, rc.Namespace),
	}

	return u.String()
}

func baseLabels(relayCore *installerv1alpha1.RelayCore) map[string]string {
	return map[string]string{
		"install.relay.sh/relay-core":  relayCore.Name,
		"app.kubernetes.io/name":       "relay-operator",
		"app.kubernetes.io/managed-by": "relay-install-operator",
	}
}
