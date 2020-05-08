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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKnativeServing(ctx context.Context, cl client.Client, version, token string) error {
	files := []string{
		"https://github.com/knative/serving/releases/download/v%s/serving-crds.yaml",
		"https://github.com/knative/serving/releases/download/v%s/serving-core.yaml",
	}

	for _, file := range files {
		vf := fmt.Sprintf(file, version)
		if err := doInstallKubernetesManifest(ctx, cl, vf, token); err != nil {
			return err
		}
	}

	return nil
}

func InstallKnativeServing(t *testing.T, ctx context.Context, cl client.Client, version string, token string) {
	require.NoError(t, doInstallKnativeServing(ctx, cl, version, token))
}

func doInstallKubernetesManifest(ctx context.Context, cl client.Client, file, token string) error {
	requested := time.Now()

	err := obj.Retry(ctx, 30*time.Second, func() *obj.RetryError {
		client := &http.Client{}
		req, err := http.NewRequest("GET", file, nil)
		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
		resp, err := client.Do(req)
		if err != nil {
			return obj.RetryTransient(err)
		} else if resp.StatusCode != http.StatusOK {
			return obj.RetryTransient(fmt.Errorf("expected 200 OK when retrieving %s configuration, got %s", file, resp.Status))
		}

		if _, err = ParseApplyKubernetesManifest(ctx, cl, resp.Body); err != nil {
			return obj.RetryPermanent(err)
		}

		return obj.RetryPermanent(nil)
	})
	if err != nil {
		return err
	}

	log.Printf("installed Knative Serving in %s", time.Now().Sub(requested))
	return nil
}
