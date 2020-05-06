package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	"github.com/puppetlabs/horsehead/v2/storage"
	_ "github.com/puppetlabs/nebula-libs/storage/file/v2"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/workflow"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/workflow/obj"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestBasic tests that an instance of the controller, when given a run to
// process, correctly sets up a Tekton pipeline and that the resulting pipeline
// should be able to access a metadata API service.
func TestBasic(t *testing.T) {
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
						rc := &rest.Config{}
						*rc = *e2e.RESTConfig
						rc.BearerToken = token
						rc.BearerTokenFile = ""

						return kubernetes.NewForConfig(rc)
					},
					middleware.KubernetesAuthenticatorWithKubernetesIntermediary(e2e.Interface),
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
			require.NoError(t, workflow.Add(deps))

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

			// Set a secret for this workflow to look up.
			vcfg.SetSecret(t, "my-tenant-id", "foo", "Hello")

			wr := &nebulav1.WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.GetName(),
					Name:      "my-test-run",
					Annotations: map[string]string{
						obj.WorkflowRunVaultSecretPathAnnotation: path.Join(vcfg.SecretsPath, "data/workflows/my-tenant-id"),
						obj.WorkflowRunDomainIDAnnotation:        "my-domain-id",
						obj.WorkflowRunTenantIDAnnotation:        "my-tenant-id",
					},
				},
				Spec: nebulav1.WorkflowRunSpec{
					Name: "my-workflow-run-1234",
					Workflow: nebulav1.Workflow{
						Parameters: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
							"Hello": "World!",
						}),
						Name: "my-workflow",
						Steps: []*nebulav1.WorkflowStep{
							{
								Name:  "my-test-step",
								Image: "alpine:latest",
								Spec: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
									"secret": map[string]interface{}{
										"$type": "Secret",
										"name":  "foo",
									},
									"param": map[string]interface{}{
										"$type": "Parameter",
										"name":  "Hello",
									},
								}),
								Input: []string{
									"trap : TERM INT",
									"sleep 600 & wait",
								},
							},
						},
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wr))

			// Wait for step to start. Could use a ListWatcher but meh.
			require.NoError(t, obj.Retry(ctx, 500*time.Millisecond, func() *obj.RetryError {
				if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
					Namespace: wr.GetNamespace(),
					Name:      wr.GetName(),
				}, wr); err != nil {
					return obj.RetryPermanent(err)
				}

				if wr.Status.Steps["my-test-step"].Status == string(obj.WorkflowRunStatusInProgress) {
					return obj.RetryPermanent(nil)
				}

				return obj.RetryTransient(fmt.Errorf("waiting for step to start"))
			}))

			// Pull the pod and get its IP.
			pod := &corev1.Pod{}
			require.NoError(t, obj.Retry(ctx, 500*time.Millisecond, func() *obj.RetryError {
				pods := &corev1.PodList{}
				if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.MatchingLabels{
					// TODO: We shouldn't really hardcode this.
					"tekton.dev/task": fmt.Sprintf("%s-%s", wr.GetName(), (&model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: "my-test-step"}).Hash().HexEncoding()),
				}); err != nil {
					return obj.RetryPermanent(err)
				}

				if len(pods.Items) == 0 {
					return obj.RetryTransient(fmt.Errorf("waiting for pod"))
				}

				pod = &pods.Items[0]
				if pod.Status.PodIP == "" {
					return obj.RetryTransient(fmt.Errorf("waiting for pod IP"))
				}

				return obj.RetryPermanent(nil)
			}))

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/spec", metadataAPI.URL), nil)
			require.NoError(t, err)
			req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var result evaluate.JSONResultEnvelope
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.True(t, result.Complete)
			assert.Equal(t, map[string]interface{}{
				"secret": "Hello",
				"param":  "World!",
			}, result.Value.Data)
		})
	})
}
