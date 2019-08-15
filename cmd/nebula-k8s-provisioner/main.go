package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/puppetlabs/horsehead/workdir"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

func main() {
	specURL := flag.String("spec-url", os.Getenv(taskutil.SpecURLEnvName), "url to fetch the spec from")
	specFile := flag.String("spec-file", "", "filepath to json formatted spec. overrides -spec-url.")
	workDir := flag.String("work-dir", "", "a working directory to store temporary and generated files")

	flag.Parse()

	log.Println("provisioning k8s cluster")

	var spec models.K8sProvisionerSpec

	if *specFile == "" {
		planOpts := taskutil.DefaultPlanOptions{SpecURL: *specURL}

		if err := taskutil.PopulateSpecFromDefaultPlan(&spec, planOpts); err != nil {
			log.Fatal(err)
		}
	} else {
		f, err := os.Open(*specFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		decoder := taskutil.DefaultJSONSpecDecoder{}

		if err := decoder.DecodeSpec(f, &spec); err != nil {
			log.Fatal(err)
		}
	}

	var wd *workdir.WorkDir

	{
		var err error
		if *workDir != "" {
			// we will NOT be calling wd.Cleanup() when using a directory passed in by a flag. This is a disaster waiting to happen.
			wd, err = workdir.New(*workDir, workdir.Options{})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			ns := workdir.NewNamespace([]string{"nebula", "task-k8s-provisioner"})
			wd, err = ns.New(workdir.DirTypeCache, workdir.Options{})
			if err != nil {
				log.Fatal(err)
			}
			// we can reliably defer the cleanup of this directory. we have used our own namespace.
			defer wd.Cleanup()
		}

	}

	manager, err := provisioning.NewK8sClusterManagerFromSpec(&spec, wd.Path)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: we need to figure out how to better provision a cluster and report readiness.
	// Currently we set a massively long timeout, which is a non-ideal solution.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	cluster, err := manager.Synchronize(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(cluster)
}
