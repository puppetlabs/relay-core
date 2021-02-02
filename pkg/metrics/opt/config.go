package opt

import (
	"io/ioutil"
	"net/http"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/metric"
)

const (
	ModuleName = "relay-metrics"

	OptionDebug          = "debug"
	OptionMetricsEnabled = "metrics_enabled"
	OptionMetricsAddress = "metrics_server_addr"

	DefaultOptionDebug          = false
	DefaultOptionMetricsEnabled = true
	DefaultOptionMetricsAddress = "0.0.0.0:3050"
)

type Config struct {
	Debug bool

	MetricsEnabled bool
	MetricsAddress string
}

func (c *Config) Metrics() (*metric.Meter, error) {
	if !c.MetricsEnabled {
		exporter, err := stdout.InstallNewPipeline([]stdout.Option{stdout.WithWriter(ioutil.Discard)}, nil)
		if err != nil {
			return nil, err
		}

		meter := exporter.MeterProvider().Meter(ModuleName)

		return &meter, nil
	}

	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{})
	if err != nil {
		return nil, err
	}
	http.HandleFunc("/", exporter.ServeHTTP)
	go func() {
		_ = http.ListenAndServe(c.MetricsAddress, nil)
	}()

	meter := exporter.MeterProvider().Meter(ModuleName)

	return &meter, nil
}

func NewConfig() (*Config, error) {
	viper.SetEnvPrefix(ModuleName)
	viper.AutomaticEnv()

	viper.SetDefault(OptionDebug, DefaultOptionDebug)
	viper.SetDefault(OptionMetricsEnabled, DefaultOptionMetricsEnabled)
	viper.SetDefault(OptionMetricsAddress, DefaultOptionMetricsAddress)

	config := &Config{
		Debug:          viper.GetBool(OptionDebug),
		MetricsEnabled: viper.GetBool(OptionMetricsEnabled),
		MetricsAddress: viper.GetString(OptionMetricsAddress),
	}

	return config, nil
}
