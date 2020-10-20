package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/horsehead/v2/mainutil"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/util/lifecycleutil"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
)

func main() {
	cfg := opt.NewConfig()

	if cfg.Debug {
		logging.SetLevel(log15.LvlDebug)
	}

	var servers []mainutil.CancelableFunc

	// TODO: Add metrics server.

	servers = append(servers, func(ctx context.Context) error {
		var auth middleware.Authenticator

		if sc, err := cfg.SampleConfig(); err != nil {
			return err
		} else if sc != nil {
			// Set up the server in sample mode for easy testing.
			var key []byte

			if ek := cfg.SampleHS256SigningKey; ek != "" {
				var err error

				key, err = base64.StdEncoding.DecodeString(ek)
				if err != nil {
					return fmt.Errorf("could not decode signing key: %+v", err)
				}
			}

			tg, err := sample.NewHS256TokenGenerator(key)
			if err != nil {
				return fmt.Errorf("failed to create token generator: %+v", err)
			}

			// Print the JWTs so users can pick them off for requests.
			_ = tg.GenerateAll(ctx, sc)

			auth = sample.NewAuthenticator(sc, tg.Key())
		} else {
			// Set up the server for real traffic.
			kc, err := cfg.KubernetesClient()
			if err != nil {
				return err
			}

			vc, err := cfg.VaultTransitClient()
			if err != nil {
				return err
			}

			auth = middleware.NewKubernetesAuthenticator(
				cfg.KubernetesClientFactory,
				middleware.KubernetesAuthenticatorWithKubernetesIntermediary(kc),
				middleware.KubernetesAuthenticatorWithChainToVaultTransitIntermediary(vc, cfg.VaultTransitPath, cfg.VaultTransitKey),
				middleware.KubernetesAuthenticatorWithVaultResolver(cfg.VaultAuthURL, cfg.VaultAuthPath, cfg.VaultAuthRole),
			)
		}

		var serverOpts []server.Option
		if cfg.Debug {
			serverOpts = append(serverOpts, server.WithErrorSensitivity(errawr.ErrorSensitivityAll))
		}

		if cfg.SentryDSN != "" {
			delegate, err := alerts.DelegateToSentry(cfg.SentryDSN)
			if err != nil {
				return fmt.Errorf("failed to initialize Sentry: %+v", err)
			}

			a := alerts.NewAlerts(delegate, alerts.Options{
				Environment: cfg.Environment,
			})

			capturer := a.NewCapturer().
				WithNewTrace().
				WithAppPackages([]string{"github.com/puppetlabs/relay-core"})

			serverOpts = append(serverOpts, server.WithCapturer(capturer))
		}

		if cfg.StepMetadataURL != "" {
			u, err := url.Parse(cfg.StepMetadataURL)
			if err != nil {
				return fmt.Errorf("failed to parse step metadata URL: %+v", err)
			}

			reg, err := validation.NewStepMetadataSchemaRegistry(u)
			if err != nil {
				return fmt.Errorf("failed to initialize step metadata schema registry: %+v", err)
			}

			serverOpts = append(serverOpts, server.WithSchemaRegistry(reg))
		}

		s := &http.Server{
			Handler: server.NewHandler(auth, serverOpts...),
			Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort),
		}

		var listenOpts []lifecycleutil.ListenWaitHTTPOption
		if cfg.TLSKeyFile != "" {
			log().Info("listening with TLS")
			listenOpts = append(listenOpts, lifecycleutil.ListenWaitWithTLS(cfg.TLSCertificateFile, cfg.TLSKeyFile))
		}

		log().Info("listening for metadata connections", "addr", s.Addr)
		return lifecycleutil.ListenWaitHTTP(ctx, s, listenOpts...)
	})

	os.Exit(mainutil.TrapAndWait(context.Background(), servers...))
}
