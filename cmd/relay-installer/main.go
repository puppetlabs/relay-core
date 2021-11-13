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

	"github.com/puppetlabs/leg/instrumentation/alerts"
	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/install/config"
	"github.com/puppetlabs/relay-core/pkg/install/controller"
	"github.com/puppetlabs/relay-core/pkg/install/dependency"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = installerv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	environment := flag.String("environment", "dev", "the environment this operator is running in")
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig file. Only required if running outside of a cluster.")
	kubeMasterURL := flag.String("kube-master-url", "", "url to the kubernetes master")
	kubeNamespace := flag.String("kube-namespace", "", "an optional working namespace to restrict to for watching CRDs")
	numWorkers := flag.Int("num-workers", 2, "the number of worker threads to spawn")
	sentryDSN := flag.String("sentry-dsn", "", "the Sentry DSN to use for error reporting")

	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// +kubebuilder:scaffold:builder

	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: *kubeMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		log.Fatal("Error creating kubernetes config", err)
	}

	var alertsDelegate alerts.DelegateFunc
	if *sentryDSN != "" {
		var err error
		alertsDelegate, err = alerts.DelegateToSentry(*sentryDSN)
		if err != nil {
			log.Fatal("Error initializing Sentry", err)
		}
	}

	cfg := &config.InstallerControllerConfig{
		Environment:             *environment,
		MaxConcurrentReconciles: *numWorkers,
		Namespace:               *kubeNamespace,
		AlertsDelegate:          alertsDelegate,
	}

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
