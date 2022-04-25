package entrypoint

import (
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type Config struct {
	DefaultTimeout time.Duration
	MetadataAPIURL *url.URL
	SecureLogging  bool
}

func NewConfig() *Config {
	conf := &Config{
		DefaultTimeout: 1 * time.Minute,
		SecureLogging:  true,
	}

	if env := os.Getenv(model.EnvironmentVariableDefaultTimeout.String()); env != "" {
		if timeout, err := time.ParseDuration(env); err == nil {
			conf.DefaultTimeout = timeout
		}
	}

	if env := os.Getenv(model.EnvironmentVariableMetadataAPIURL.String()); env != "" {
		if u, err := url.Parse(env); err == nil {
			conf.MetadataAPIURL = u
		}
	}

	if env := os.Getenv(model.EnvironmentVariableEnableSecureLogging.String()); env != "" {
		if secureLogging, err := strconv.ParseBool(env); err == nil {
			conf.SecureLogging = secureLogging
		}
	}

	return conf
}
