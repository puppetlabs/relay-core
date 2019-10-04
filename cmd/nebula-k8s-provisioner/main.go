package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/puppetlabs/horsehead/workdir"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/client"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
)

func main() {
	specURL := flag.String("spec-url", os.Getenv(taskutil.SpecURLEnvName), "url to fetch the spec from")
	workDir := flag.String("work-dir", "", "a working directory to store temporary and generated files")

	flag.Parse()

	log.Println("provisioning k8s cluster")

	planOpts := taskutil.DefaultPlanOptions{SpecURL: *specURL}

	var spec models.K8sProvisionerSpec
	if err := taskutil.PopulateSpecFromDefaultPlan(&spec, planOpts); err != nil {
		log.Fatal(err)
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

	outputs, err := client.NewDefaultOutputsClientFromNebulaEnv()
	if err != nil {
		log.Fatal(err)
	}

	managerCfg := provisioning.K8sClusterManagerConfig{
		Spec:          &spec,
		Workdir:       wd.Path,
		OutputsClient: outputs,
	}

	manager, err := provisioning.NewK8sClusterManagerFromSpec(managerCfg)
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
