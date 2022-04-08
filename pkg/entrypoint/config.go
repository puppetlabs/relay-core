package entrypoint

import (
	"net/url"
	"os"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type Config struct {
	DeploymentEnvironment *model.DeploymentEnvironment
	MetadataAPIURL        *url.URL
}

func NewConfig() *Config {
	conf := &Config{
		DeploymentEnvironment: &model.DeploymentEnvironmentDefault,
	}

	if env := os.Getenv(model.EnvironmentVariableDeploymentEnvironment.String()); env != "" {
		if environment, ok := model.DeploymentEnvironments[env]; ok {
			conf.DeploymentEnvironment = &environment
		}
	}

	if env := os.Getenv(model.EnvironmentVariableMetadataAPIURL.String()); env != "" {
		if u, err := url.Parse(env); err == nil {
			conf.MetadataAPIURL = u
		}
	}

	return conf
}
