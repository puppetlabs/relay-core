package config

import "github.com/spf13/viper"

type WebhookControllerConfig struct {
	Debug                            bool
	Name                             string
	Namespace                        string
	ServiceName                      string
	CertificateSecretName            string
	MutatingWebhookConfigurationName string
}

func NewWebhookControllerConfig(defaultName string) *WebhookControllerConfig {
	viper.SetEnvPrefix("relay_operator")
	viper.AutomaticEnv()

	viper.SetDefault("name", defaultName)

	return &WebhookControllerConfig{
		Debug:                            viper.GetBool("debug"),
		Name:                             viper.GetString("name"),
		Namespace:                        viper.GetString("namespace"),
		ServiceName:                      viper.GetString("service_name"),
		CertificateSecretName:            viper.GetString("certificate_secret_name"),
		MutatingWebhookConfigurationName: viper.GetString("mutating_webhook_configuration_name"),
	}
}
