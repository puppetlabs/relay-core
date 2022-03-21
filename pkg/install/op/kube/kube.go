package kube

import (
	"github.com/google/wire"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ProviderSet = wire.NewSet(
	NewKubeScheme,
	NewKubeClient,
)

func NewKubeScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		kubernetesscheme.AddToScheme,
		metav1.AddMetaToScheme,
		rbacv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		apiextensionsv1beta1.AddToScheme,
	)
	_ = schemeBuilder.AddToScheme(scheme)
	return scheme
}

func NewKubeClient(scheme *runtime.Scheme) (client.Client, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{
		Scheme: scheme,
	})
}
