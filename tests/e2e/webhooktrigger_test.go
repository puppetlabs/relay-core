package e2e_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	"github.com/puppetlabs/horsehead/v2/storage"
	_ "github.com/puppetlabs/nebula-libs/storage/file/v2"
	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/tenant"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/trigger"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	"github.com/puppetlabs/nebula-tasks/pkg/util/retry"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWebhookTrigger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testutil.WithVault(t, func(vcfg *testutil.Vault) {
		e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
			mets, err := metrics.NewNamespace("workflow_controller", metrics.Options{
				DelegateType:  delegates.NoopDelegate,
				ErrorBehavior: metrics.ErrorBehaviorLog,
			})
			require.NoError(t, err)

			tmp, err := ioutil.TempDir("", "relay-e2e-")
			require.NoError(t, err)
			defer os.RemoveAll(tmp)

			blobStore, err := storage.NewBlobStore(url.URL{Scheme: "file", Path: tmp})
			require.NoError(t, err)

			imagePullSecret := obj.NewImagePullSecret(client.ObjectKey{
				Namespace: ns.GetName(),
				Name:      "test",
			})
			imagePullSecret.Object.Data = map[string][]byte{
				".dockerconfigjson": []byte(`{}`),
			}
			require.NoError(t, imagePullSecret.Persist(ctx, e2e.ControllerRuntimeClient))

			metadataAPI := httptest.NewServer(server.NewHandler(
				middleware.NewKubernetesAuthenticator(
					func(token string) (kubernetes.Interface, error) {
						rc := rest.AnonymousClientConfig(e2e.RESTConfig)
						rc.BearerToken = token
						rc.BearerTokenFile = ""

						return kubernetes.NewForConfig(rc)
					},
					middleware.KubernetesAuthenticatorWithKubernetesIntermediary(&authenticate.KubernetesInterface{
						Interface:       e2e.Interface,
						TektonInterface: e2e.TektonInterface,
					}),
					middleware.KubernetesAuthenticatorWithChainToVaultTransitIntermediary(vcfg.Client, vcfg.TransitPath, vcfg.TransitKey),
					middleware.KubernetesAuthenticatorWithVaultResolver(vcfg.Address, vcfg.JWTAuthPath, vcfg.JWTAuthRole),
				),
				server.WithTrustedProxyHops(1),
			))
			defer metadataAPI.Close()

			metadataAPIURL, err := url.Parse(metadataAPI.URL)
			require.NoError(t, err)

			cfg := &config.WorkflowControllerConfig{
				Namespace:               ns.GetName(),
				ImagePullSecret:         imagePullSecret.Key.Name,
				MaxConcurrentReconciles: 16,
				MetadataAPIURL:          metadataAPIURL,
				VaultTransitPath:        vcfg.TransitPath,
				VaultTransitKey:         vcfg.TransitKey,
			}

			deps, err := dependency.NewDependencyManager(cfg, e2e.RESTConfig, vcfg.Client, vcfg.JWTSigner, blobStore, mets)
			require.NoError(t, err)
			require.NoError(t, tenant.Add(deps.Manager, cfg))
			require.NoError(t, trigger.Add(deps))

			var wg sync.WaitGroup

			ch := make(chan struct{})

			wg.Add(1)
			go func() {
				defer wg.Done()
				require.NoError(t, deps.Manager.Start(ch))
			}()
			defer func() {
				close(ch)
				wg.Wait()
			}()

			// Set a secret and connection for this workflow to look up.
			vcfg.SetSecret(t, "my-tenant-id", "foo", "Hello")
			vcfg.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "very-nice-key",
			})

			tn := &relayv1beta1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-tenant-id",
					Namespace: ns.GetName(),
				},
				Spec: relayv1beta1.TenantSpec{
					NamespaceTemplate: relayv1beta1.NamespaceTemplate{
						Metadata: metav1.ObjectMeta{
							Name: "my-tenant",
							Labels: map[string]string{
								"my-tenant-label-name": "my-tenant-label-value",
							},
						},
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))

			// Wait for step to start. Could use a ListWatcher but meh.
			require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: tn.GetNamespace(),
					Name:      tn.GetName(),
				}, tn); err != nil {
					return retry.RetryPermanent(err)
				}

				tc := relayv1beta1.TenantCondition{}
				tnsc := relayv1beta1.TenantCondition{}
				for _, condition := range tn.Status.Conditions {
					switch condition.Type {
					case relayv1beta1.TenantReady:
						tc = condition
					case relayv1beta1.TenantNamespaceReady:
						tnsc = condition
					}

					if tc.Condition.Status == corev1.ConditionTrue &&
						tnsc.Condition.Status == corev1.ConditionTrue {
						return retry.RetryPermanent(nil)
					}
				}

				return retry.RetryTransient(fmt.Errorf("waiting for tenant to be successfully created"))
			}))

			wt := &relayv1beta1.WebhookTrigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-trigger",
					Namespace: ns.GetName(),
					Labels: map[string]string{
						"my-trigger-label-name": "my-trigger-label-value",
					},
					Annotations: map[string]string{
						model.RelayVaultEngineMountAnnotation:    vcfg.SecretsPath,
						model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
						model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
						model.RelayDomainIDAnnotation:            "my-domain-id",
						model.RelayTenantIDAnnotation:            "my-tenant-id",
					},
				},
				Spec: relayv1beta1.WebhookTriggerSpec{
					Image: "gcr.io/nebula-tasks/coalsack-workflow-receiver",
					TenantRef: corev1.LocalObjectReference{
						Name: "my-tenant-id",
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

			// Wait for step to start. Could use a ListWatcher but meh.
			require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: wt.GetNamespace(),
					Name:      wt.GetName(),
				}, wt); err != nil {
					return retry.RetryPermanent(err)
				}

				if wt.Status.URL == "" {
					return retry.RetryTransient(fmt.Errorf("waiting for webhook trigger to be successfully created"))
				}

				return retry.RetryPermanent(nil)
			}))

			kns := &servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-trigger",
					Namespace: ns.GetName(),
				},
			}

			// Wait for step to start. Could use a ListWatcher but meh.
			require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: kns.GetNamespace(),
					Name:      kns.GetName(),
				}, kns); err != nil {
					return retry.RetryPermanent(err)
				}

				actual := map[apis.ConditionType]apis.Condition{}
				for _, condition := range kns.Status.Conditions {
					actual[condition.Type] = condition
				}

				for _, ct := range []apis.ConditionType{servingv1.ServiceConditionReady, servingv1.ServiceConditionRoutesReady, servingv1.ServiceConditionConfigurationsReady} {
					condition, ok := actual[ct]
					if !ok {
						return retry.RetryTransient(fmt.Errorf("did not find expected knative service condition"))
					}

					if condition.Status == corev1.ConditionFalse {
						return retry.RetryTransient(fmt.Errorf("knative service condition is false"))
					}

					// FIXME Allow IngressNotConfigured (for now, until an appropriate ingress controller (i.e. Ambassador) is installed in the testing cluster)
					if condition.Status == corev1.ConditionUnknown {
						if condition.Reason != "IngressNotConfigured" {
							return retry.RetryTransient(fmt.Errorf("knative service condition is unknown"))
						}
					}
				}

				return retry.RetryPermanent(nil)
			}))
		})
	})
}
