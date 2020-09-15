package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKubernetesManifest(ctx context.Context, cl client.Client, pattern string, patchers ...ParseKubernetesManifestPatcherFunc) error {
	files, err := getFixtures(pattern)
	if err != nil {
		return err
	}

	for _, file := range files {
		log.Printf("applying manifest %s", file)

		reader, err := os.Open(file)
		if err != nil {
			return err
		}

		if _, err := ParseApplyKubernetesManifest(ctx, cl, reader, patchers...); err != nil {
			return err
		}
	}

	return nil
}

func doInstall(ctx context.Context, cl client.Client, namespace, name string, patchers ...ParseKubernetesManifestPatcherFunc) error {
	requested := time.Now()

	pattern := fmt.Sprintf("fixtures/%s/*.yaml", name)
	err := doInstallKubernetesManifest(ctx, cl, pattern, patchers...)
	if err != nil {
		return err
	}

	log.Printf("installed %s in %s after %s", name, namespace, time.Now().Sub(requested))
	return nil
}

func doInstallAndWait(ctx context.Context, cl client.Client, namespace, name string, patchers ...ParseKubernetesManifestPatcherFunc) error {
	doInstall(ctx, cl, namespace, name, patchers...)

	requested := time.Now()

	err := WaitForServicesToBeReady(ctx, cl, namespace)
	if err != nil {
		return err
	}

	log.Printf("waited for services to be ready for %s in %s after %s", name, namespace, time.Now().Sub(requested))
	return nil
}
