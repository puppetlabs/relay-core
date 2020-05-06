package testutil

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallTektonPipeline(ctx context.Context, cl client.Client, version string) error {
	resp, err := http.Get(fmt.Sprintf("https://storage.googleapis.com/tekton-releases/pipeline/previous/v%s/release.yaml", version))
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200 OK when retrieving Tekton configuration, got %s", resp.Status)
	}

	_, err = ParseApplyKubernetesManifest(ctx, cl, resp.Body)
	return err
}

func InstallTektonPipeline(t *testing.T, ctx context.Context, cl client.Client, version string) {
	require.NoError(t, doInstallTektonPipeline(ctx, cl, version))
}
