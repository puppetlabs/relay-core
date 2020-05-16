package e2e_test

import (
	"context"
	"fmt"
	"io/ioutil"
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
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/tenant"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/trigger"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/util/retry"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWebhookTriggerServesResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

			cfg := &config.WorkflowControllerConfig{
				Namespace:               ns.GetName(),
				MaxConcurrentReconciles: 16,
				MetadataAPIURL:          &url.URL{Scheme: "http", Host: "localhost"},
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

			tn := &relayv1beta1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-tenant",
					Namespace: ns.GetName(),
				},
				Spec: relayv1beta1.TenantSpec{
					NamespaceTemplate: relayv1beta1.NamespaceTemplate{
						Metadata: metav1.ObjectMeta{
							Name: fmt.Sprintf("%s-child", ns.GetName()),
						},
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				require.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, tn))
				require.NoError(t, testutil.WaitForObjectDeletion(ctx, e2e.ControllerRuntimeClient, tn))
			}()

			// Wait for TenantReady.
			require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: tn.GetNamespace(),
					Name:      tn.GetName(),
				}, tn); err != nil {
					return retry.RetryPermanent(err)
				}

				for _, cond := range tn.Status.Conditions {
					if cond.Type != relayv1beta1.TenantReady {
						continue
					} else if cond.Status != corev1.ConditionTrue {
						break
					}

					return retry.RetryPermanent(nil)
				}

				return retry.RetryTransient(fmt.Errorf("waiting for tenant to be successfully created"))
			}))

			// Create a trigger.
			wt := &relayv1beta1.WebhookTrigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-trigger",
					Namespace: ns.GetName(),
				},
				Spec: relayv1beta1.WebhookTriggerSpec{
					Image: "hashicorp/http-echo",
					Args: []string{
						"-listen", ":8080",
						"-text", "Hello, Relay!",
					},
					TenantRef: corev1.LocalObjectReference{
						Name: tn.GetName(),
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				require.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, wt))
				require.NoError(t, testutil.WaitForObjectDeletion(ctx, e2e.ControllerRuntimeClient, wt))
			}()

			// Wait for trigger to settle in Knative and pull its URL.
			var targetURL string
			require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: wt.GetNamespace(),
					Name:      wt.GetName(),
				}, wt); err != nil {
					return retry.RetryPermanent(err)
				}

				for _, cond := range wt.Status.Conditions {
					if cond.Type != relayv1beta1.WebhookTriggerReady {
						continue
					} else if cond.Status != corev1.ConditionTrue {
						break
					}

					targetURL = wt.Status.URL
					return retry.RetryPermanent(nil)
				}

				return retry.RetryTransient(fmt.Errorf("waiting for webhook trigger to be successfully created"))
			}))
			require.NotEmpty(t, targetURL)

			code, stdout, stderr := testutil.RunScriptInAlpine(
				t, ctx, e2e.RESTConfig, e2e.Interface,
				metav1.ObjectMeta{
					Namespace: ns.GetName(),
				},
				fmt.Sprintf("exec wget -q -O - %s", targetURL),
			)
			assert.Equal(t, 0, code, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", stdout, stderr)
			assert.Contains(t, stdout, "Hello, Relay!")
		})
	})
}
