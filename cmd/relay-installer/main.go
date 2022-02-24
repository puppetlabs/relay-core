/*
Copyright 2020 Puppet, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"log"
	"os"

	"github.com/puppetlabs/leg/instrumentation/alerts"
	"github.com/puppetlabs/relay-core/pkg/install/config"
	"github.com/puppetlabs/relay-core/pkg/install/controller"
	"github.com/puppetlabs/relay-core/pkg/install/dependency"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var setupLog = ctrl.Log.WithName("setup")

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	var (
		metricsAddr          string
		enableLeaderElection bool
		environment          string
		kubeconfig           string
		kubeMasterURL        string
		kubeNamespace        string
		numWorkers           int
		sentryDSN            string
	)

	fs.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	fs.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.StringVar(&environment, "environment", "dev", "the environment this operator is running in")
	fs.StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file. Only required if running outside of a cluster.")
	fs.StringVar(&kubeMasterURL, "kube-master-url", "", "url to the kubernetes master")
	fs.StringVar(&kubeNamespace, "kube-namespace", "", "an optional working namespace to restrict to for watching CRDs")
	fs.IntVar(&numWorkers, "num-workers", 2, "the number of worker threads to spawn")
	fs.StringVar(&sentryDSN, "sentry-dsn", "", "the Sentry DSN to use for error reporting")

	fs.Parse(os.Args[1:])

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: kubeMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		log.Fatal("Error creating kubernetes config", err)
	}

	alertsDelegate, _ := alerts.DelegateToPassthrough()
	if sentryDSN != "" {
		var err error
		alertsDelegate, err = alerts.DelegateToSentry(sentryDSN)
		if err != nil {
			log.Fatal("Error initializing Sentry", err)
		}
	}

	cfg := &config.InstallerControllerConfig{
		Environment:             environment,
		MaxConcurrentReconciles: numWorkers,
		Namespace:               kubeNamespace,
		AlertsDelegate:          alertsDelegate,
	}

	setupLog.Info("creating manager")
	m, err := dependency.NewManager(cfg, kcc)
	if err != nil {
		log.Fatal("Error creating controller dependency builder", err)
	}

	if err := controller.Add(m); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	setupLog.Info("starting manager")
	if err := m.Manager.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal("Manager exited non-zero", err)
	}
}
