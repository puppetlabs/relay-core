package testutil

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/reconciler/workflow/obj"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallTektonPipeline(ctx context.Context, cl client.Client, version string) error {
	requested := time.Now()

	resp, err := http.Get(fmt.Sprintf("https://storage.googleapis.com/tekton-releases/pipeline/previous/v%s/release.yaml", version))
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200 OK when retrieving Tekton configuration, got %s", resp.Status)
	}

	if _, err := ParseApplyKubernetesManifest(ctx, cl, resp.Body); err != nil {
		return err
	}

	// Wait for Tekton services to be ready.
	err = obj.Retry(ctx, 2*time.Second, func() *obj.RetryError {
		eps := &corev1.EndpointsList{}
		if err := cl.List(ctx, eps, client.InNamespace("tekton-pipelines")); err != nil {
			return obj.RetryPermanent(err)
		}

		if len(eps.Items) == 0 {
			return obj.RetryTransient(fmt.Errorf("waiting for endpoints"))
		}

		for _, ep := range eps.Items {
			log.Println("checking Tekton service", ep.Name)
			if len(ep.Subsets) == 0 {
				return obj.RetryTransient(fmt.Errorf("waiting for subsets"))
			}

			for _, subset := range ep.Subsets {
				if len(subset.Addresses) == 0 {
					return obj.RetryTransient(fmt.Errorf("waiting for pod assignment"))
				}
			}
		}

		return obj.RetryPermanent(nil)
	})
	if err != nil {
		return err
	}

	log.Printf("installed Tekton Pipeline in %s", time.Now().Sub(requested))
	return nil
}

func InstallTektonPipeline(t *testing.T, ctx context.Context, cl client.Client, version string) {
	require.NoError(t, doInstallTektonPipeline(ctx, cl, version))
}
