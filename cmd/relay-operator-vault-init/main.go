package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
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

var (
	RuleIsConnectionRefused = errmark.RuleFunc(IsConnectionRefused)
)

func IsConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connect: connection refused")
}

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

	err = retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := vc.InitializeVault(ctx); err != nil {
			retryOnError := false
			errmark.If(err, RuleIsConnectionRefused, func(err error) {
				retryOnError = true
			})

			if retryOnError {
				return retry.Repeat(err)
			}

			return retry.Done(err)
		}
		return retry.Done(nil)
	}, retry.WithBackoffFactory(
		backoff.Build(
			backoff.Exponential(500*time.Microsecond, 2),
			backoff.MaxBound(30*time.Second),
			backoff.FullJitter(),
			backoff.MaxRetries(20),
			backoff.NonSliding,
		),
	))
	if err != nil {
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
