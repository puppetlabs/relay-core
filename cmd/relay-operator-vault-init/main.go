package main

import (
	"context"
	"log"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
	"github.com/puppetlabs/relay-core/pkg/install/vault"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultScheme = runtime.NewScheme()
	schemeBuilder = runtime.NewSchemeBuilder(
		kubernetesscheme.AddToScheme,
		metav1.AddMetaToScheme,
		rbacv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		apiextensionsv1beta1.AddToScheme,
	)
	_ = schemeBuilder.AddToScheme(DefaultScheme)
)

type services struct {
	vault *vaultapi.Client
}

func main() {
	ctx := context.Background()
	vaultConfig, vaultCoreConfig, err := vault.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	svcs, err := InitializeServices(ctx, vaultConfig)
	if err != nil {
		log.Fatal(err)
	}

	kubeClient, err := NewKubeClient(DefaultScheme)
	if err != nil {
		log.Fatal(err)
	}

	vsc := vaultutil.NewVaultSystemManager(svcs.vault, kubeClient, vaultConfig)

	vc := vault.NewVaultInitializer(vsc, vaultConfig, vaultCoreConfig)

	if err := vc.InitializeVault(ctx); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
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
