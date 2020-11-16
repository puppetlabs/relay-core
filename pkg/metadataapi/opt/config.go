package opt

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	yaml "gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

const (
	DefaultListenPort      = 7000
	DefaultVaultURL        = "http://localhost:8200"
	DefaultStepMetadataURL = "https://relay.sh/step-metadata.json"

	DefaultKubernetesAutomountTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	DefaultKubernetesAutomountCAFile    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type Config struct {
	// Debug determines whether this server starts with debugging enabled.
	Debug bool

	// Environment is the execution environment for this instance. Used for
	// reporting errors.
	Environment string

	// ListenPort is the port to bind the server to.
	ListenPort int

	// TLSKeyFile is the relative path to a PEM-encoded X509 private key to
	// enable TLS.
	TLSKeyFile string

	// TLSCertFile is the relative path to a PEM-encoded X509 certificate
	// bundle corresponding to the private key.
	TLSCertificateFile string

	// LogServiceURL is the HTTP(S) url to the log service
	LogServiceURL string

	// StepMetadataURL is the HTTP(S) url to the relaysh core step metadata
	// json file.
	StepMetadataURL string

	// VaultTransitURL is the HTTP(S) URL to the Vault server to use for secure
	// token decryption.
	VaultTransitURL string

	// VaultTransitToken is the token to use to authenticate transit requests.
	VaultTransitToken string

	// VaultTransitPath is the path to the transit secrets engine providing
	// token decryption.
	VaultTransitPath string

	// VaultTransitKey is the key to use for token decryption.
	VaultTransitKey string

	// VaultAuthURL is the HTTP(S) URL to the Vault server to use for
	// authenticating tenants.
	VaultAuthURL string

	// VaultAuthPath is the path to the JWT secrets engine providing
	// authentication for tenants.
	VaultAuthPath string

	// VaultAuthRole is the role to use when logging in as a tenant.
	VaultAuthRole string

	// KubernetesURL is the the HTTP(S) URL to the Kubernetes cluster master.
	KubernetesURL string

	// KubernetesCAData is certificate authority data for the Kubernetes cluster
	// to connect to.
	KubernetesCAData string

	// KubernetesServiceAccountToken is the service account token to use for
	// reading pod data.
	KubernetesServiceAccountToken string

	// SampleConfigFiles is a list of configuration files that configure this
	// instance of the metadata API to serve sample data for demo or testing
	// purposes.
	SampleConfigFiles []string

	// SampleHS256SigningKey is a base64-encoded signing key for handling JWTs
	// from sample steps.
	SampleHS256SigningKey string

	// SentryDSN is an optional identifier to automatically log API errors to
	// Sentry.
	SentryDSN string
}

func (c *Config) kubernetesInClusterHost() (string, bool) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return "", false
	}

	return (&url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}).String(), true
}

func (c *Config) kubernetesClientConfig() (*rest.Config, error) {
	inClusterHost, inCluster := c.kubernetesInClusterHost()

	cfg := &rest.Config{
		Host: c.KubernetesURL,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte(c.KubernetesCAData),
		},
	}

	if cfg.Host == "" {
		if !inCluster {
			return nil, rest.ErrNotInCluster
		}

		cfg.Host = inClusterHost
	}

	if len(cfg.TLSClientConfig.CAData) == 0 {
		if !inCluster {
			return nil, rest.ErrNotInCluster
		} else if _, err := cert.NewPool(DefaultKubernetesAutomountCAFile); err != nil {
			return nil, err
		}

		cfg.CAFile = DefaultKubernetesAutomountCAFile
	}

	return cfg, nil
}

func (c *Config) KubernetesClientFactory(token string) (kubernetes.Interface, error) {
	cfg, err := c.kubernetesClientConfig()
	if err != nil {
		return nil, err
	}

	cfg.BearerToken = token

	return kubernetes.NewForConfig(cfg)
}

func (c *Config) KubernetesClient() (*authenticate.KubernetesInterface, error) {
	cfg, err := c.kubernetesClientConfig()
	if err != nil {
		return nil, err
	}

	cfg.BearerToken = c.KubernetesServiceAccountToken
	if cfg.BearerToken == "" {
		if _, ok := c.kubernetesInClusterHost(); !ok {
			return nil, rest.ErrNotInCluster
		}

		// Attempt to read from standard file.
		b, err := ioutil.ReadFile(DefaultKubernetesAutomountTokenFile)
		if err != nil {
			return nil, err
		}

		cfg.BearerToken = string(b)
		cfg.BearerTokenFile = DefaultKubernetesAutomountTokenFile
	}

	return authenticate.NewKubernetesInterfaceForConfig(cfg)
}

func (c *Config) LogServiceClient() (plspb.LogClient, error) {
	if c.LogServiceURL == "" {
		return nil, nil
	}

	conn, err := grpc.Dial(c.LogServiceURL, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return plspb.NewLogClient(conn), nil
}

func (c *Config) VaultTransitClient() (*vaultapi.Client, error) {
	// Transit is authoritative so can safely fall back to the default config.
	cfg := vaultapi.DefaultConfig()
	cfg.Address = c.VaultTransitURL

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client.SetToken(c.VaultTransitToken)

	return client, nil
}

func (c *Config) SampleConfig() (*SampleConfig, error) {
	if len(c.SampleConfigFiles) == 0 {
		return nil, nil
	}

	sc := &SampleConfig{
		Connections: make(SampleConfigConnections),
		Secrets:     make(map[string]string),
		Runs:        make(map[string]*SampleConfigRun),
		Triggers:    make(map[string]*SampleConfigTrigger),
	}

	for _, f := range c.SampleConfigFiles {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("opt: cannot read sample file %q: %+v", f, err)
		}

		var tsc SampleConfig
		if err := yaml.Unmarshal(b, &tsc); err != nil {
			return nil, fmt.Errorf("opt: cannot parse sample file %q: %+v", f, err)
		}

		tsc.AppendTo(sc)
	}

	return sc, nil
}

func NewConfig() *Config {
	viper.SetEnvPrefix("relay_metadata_api")
	viper.AutomaticEnv()

	viper.BindEnv("vault_addr", "VAULT_ADDR")
	viper.SetDefault("vault_addr", DefaultVaultURL)

	viper.BindEnv("vault_token", "VAULT_TOKEN")

	viper.SetDefault("environment", "dev")
	viper.SetDefault("listen_port", DefaultListenPort)

	viper.SetDefault("vault_transit_url", viper.GetString("vault_addr"))
	viper.SetDefault("vault_transit_token", viper.GetString("vault_token"))
	viper.SetDefault("vault_transit_path", "transit")
	viper.SetDefault("vault_transit_key", "metadata-api")

	viper.SetDefault("vault_auth_url", viper.GetString("vault_addr"))
	viper.SetDefault("vault_auth_path", "auth/jwt")

	viper.SetDefault("step_metadata_url", DefaultStepMetadataURL)

	return &Config{
		Debug:       viper.GetBool("debug"),
		Environment: viper.GetString("environment"),
		ListenPort:  viper.GetInt("listen_port"),

		TLSKeyFile:         viper.GetString("tls_key_file"),
		TLSCertificateFile: viper.GetString("tls_certificate_file"),

		LogServiceURL: viper.GetString("log_service_url"),

		StepMetadataURL: viper.GetString("step_metadata_url"),

		VaultTransitURL:   viper.GetString("vault_transit_url"),
		VaultTransitToken: viper.GetString("vault_transit_token"),
		VaultTransitPath:  viper.GetString("vault_transit_path"),
		VaultTransitKey:   viper.GetString("vault_transit_key"),

		VaultAuthURL:  viper.GetString("vault_auth_url"),
		VaultAuthPath: viper.GetString("vault_auth_path"),
		VaultAuthRole: viper.GetString("vault_auth_role"),

		KubernetesURL:                 viper.GetString("kubernetes_url"),
		KubernetesCAData:              viper.GetString("kubernetes_ca_data"),
		KubernetesServiceAccountToken: viper.GetString("kubernetes_service_account_token"),

		SampleConfigFiles:     viper.GetStringSlice("sample_config_files"),
		SampleHS256SigningKey: viper.GetString("sample_hs256_signing_key"),

		SentryDSN: viper.GetString("sentry_dsn"),
	}
}
