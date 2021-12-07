package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/puppetlabs/leg/k8sutil/pkg/app/selfsignedsecret"
	"github.com/puppetlabs/leg/k8sutil/pkg/app/webhookcert"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/eventctx"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/mainutil"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var schemes = runtime.NewSchemeBuilder(
	scheme.AddToScheme,
)

func main() {
	cfg := config.NewWebhookControllerConfig("relay-operator")

	secretKey := client.ObjectKey{
		Name:      cfg.CertificateSecretName,
		Namespace: cfg.Namespace,
	}

	mgrOpts := manager.Options{
		LeaderElection:                false,
		LeaderElectionID:              fmt.Sprintf("%s.lease.operator-webhook-certificate.relay.sh", cfg.Name),
		LeaderElectionResourceLock:    resourcelock.LeasesResourceLock,
		LeaderElectionReleaseOnCancel: true,
		LeaderElectionNamespace:       cfg.Namespace,
		Namespace:                     cfg.Namespace,
	}

	transforms := []func(manager.Manager) error{
		func(mgr manager.Manager) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			secret := corev1.NewTLSSecret(secretKey)
			if err := secret.Persist(ctx, mgr.GetClient()); errors.IsAlreadyExists(err) {
				fmt.Println(err)
				return nil
			} else if err != nil {
				fmt.Println(err)
				return err
			}

			return nil
		},
		func(mgr manager.Manager) error {
			return webhookcert.AddReconcilerToManager(
				mgr,
				secretKey,
				webhookcert.WithMutatingWebhookConfiguration(cfg.MutatingWebhookConfigurationName),
			)
		},
		func(mgr manager.Manager) error {
			return selfsignedsecret.AddReconcilerToManager(
				mgr,
				secretKey,
				"Puppet, Inc.",
				fmt.Sprintf("%s.%s.svc", cfg.ServiceName, cfg.Namespace),
			)
		},
	}

	os.Exit(mainutil.TrapAndWait(context.Background(), func(ctx context.Context) error {
		defer klog.Flush()

		flag.Parse()

		kfs := flag.NewFlagSet("klog", flag.ExitOnError)
		klog.InitFlags(kfs)

		if cfg.Debug {
			_ = kfs.Set("v", "5")
		}

		log.SetLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))

		if mgrOpts.Scheme == nil {
			s := runtime.NewScheme()
			if err := schemes.AddToScheme(s); err != nil {
				return fmt.Errorf("failed to create scheme: %w", err)
			}

			mgrOpts.Scheme = s
		}

		mgr, err := manager.New(crconfig.GetConfigOrDie(), mgrOpts)
		if err != nil {
			return fmt.Errorf("failed to create manager: %w", err)
		}

		for i, transform := range transforms {
			if err := transform(mgr); err != nil {
				return fmt.Errorf("failed to apply manager transform #%d: %w", i, err)
			}
		}

		return mgr.Start(eventctx.WithEventRecorder(ctx, mgr, cfg.Name))
	}))
}
