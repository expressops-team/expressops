package config

import (
	"context"
	"fmt"
	"os"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	// We import the LoadPlugin and GetPlugin functions from the pluginManager package
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Load the configuration from YAML

func LoadConfig(ctx context.Context, path string, logger *logrus.Logger) (*v1alpha1.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// declares a variable of type Config
	// with the yaml package("gopkg.in/yaml.v3"), we unmarshal the data into the cfg variable

	var cfg v1alpha1.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error al deserializar YAML: %w", err)
	}

	logger.Info("Configuración base cargada. Procesando plugins e inyectando secretos...")
	// for each plugin in the config, we load the plugin, get the plugin instance and initialize it

	for i := range cfg.Plugins {

		pluginCfg := &cfg.Plugins[i]

		if pluginCfg.Name == "slack-notifier" {
			webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
			if webhookURL == "" {

				msg := fmt.Sprintf("la variable de entorno SLACK_WEBHOOK_URL es requerida por el plugin '%s' pero no está definida", pluginCfg.Name)
				logger.Error(msg)
				return nil, fmt.Errorf("%s", msg)
			}

			if pluginCfg.Config == nil {
				pluginCfg.Config = make(map[string]interface{})
			}

			pluginCfg.Config["webhook_url"] = webhookURL
			logger.Debugf("Inyectada SLACK_WEBHOOK_URL en la configuración del plugin '%s'", pluginCfg.Name)
		}

		logger.Debugf("Cargando código del plugin: %s (Path: %s)", pluginCfg.Name, pluginCfg.Path)
		if err := pluginManager.LoadPlugin(ctx, pluginCfg.Path, pluginCfg.Name, pluginCfg.Config, logger); err != nil {

			return nil, fmt.Errorf("error cargando el código/instancia del plugin '%s' desde '%s': %w", pluginCfg.Name, pluginCfg.Path, err)
		}
		logger.Infof("Plugin '%s' procesado correctamente.", pluginCfg.Name)
	}

	logger.Info("Todos los plugins procesados. Configuración final lista.")
	return &cfg, nil
}
