package testutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKubernetesManifest(ctx context.Context, cl client.Client, mapper meta.RESTMapper, namespace, pattern string) error {
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

		if _, err := ParseApplyKubernetesManifest(ctx, cl, mapper, namespace, reader); err != nil {
			return err
		}
	}

	return nil
}

func doInstall(ctx context.Context, cl client.Client, mapper meta.RESTMapper, namespace, name, version string) error {
	requested := time.Now()

	pattern := fmt.Sprintf("fixtures/%s/*-v%s-*.yaml", name, version)
	err := doInstallKubernetesManifest(ctx, cl, mapper, namespace, pattern)
	if err != nil {
		return err
	}

	err = WaitForServicesToBeReady(ctx, cl, namespace)
	if err != nil {
		return err
	}

	log.Printf("installed %s in %s after %s", name, namespace, time.Now().Sub(requested))
	return nil
}
