package opt

import (
	"io"
	"net/http"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/metric"
)

const (
	ModuleName = "relay_metrics"

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
		_, exporter, err := stdout.InstallNewPipeline([]stdout.Option{stdout.WithWriter(io.Discard)}, nil)
		if err != nil {
			return nil, err
		}

		meter := exporter.MeterProvider().Meter(ModuleName)

		return &meter, nil
	}

	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{
			// TODO: Once views are implemented into OpenTelemetry, we should
			// define this in the spec. In the meantime, this is used to bucket
			// timings, so we'll use seconds-based histograms.
			//
			// https://github.com/open-telemetry/oteps/pull/89
			0, 5, 10, 20, 30, 45, 60, 120, 300, 600, 1800, 3600,
		},
	})
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
