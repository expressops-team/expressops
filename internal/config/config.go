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
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	logger.Info("Base configuration loaded. Processing plugins...")

	// Process each plugin in the configuration
	for i := range cfg.Plugins {
		pluginCfg := &cfg.Plugins[i]

		// Process environment variables for plugin configuration
		for key, value := range pluginCfg.Config {
			if strValue, ok := value.(string); ok && len(strValue) > 0 && strValue[0] == '$' {
				envVarName := strValue[1:]
				envVarValue := os.Getenv(envVarName)

				if envVarValue == "" {
					msg := fmt.Sprintf("environment variable %s required by plugin '%s' is not defined", envVarName, pluginCfg.Name)
					logger.Error(msg)
					return nil, fmt.Errorf("%s", msg)
				}

				pluginCfg.Config[key] = envVarValue
				logger.Debugf("Injected %s environment variable into plugin '%s' configuration", envVarName, pluginCfg.Name)
			}
		}

		logger.Debugf("Loading plugin code: %s (Path: %s)", pluginCfg.Name, pluginCfg.Path)
		if err := pluginManager.LoadPlugin(ctx, pluginCfg.Path, pluginCfg.Name, pluginCfg.Config, logger); err != nil {
			return nil, fmt.Errorf("error loading plugin '%s' from '%s': %w", pluginCfg.Name, pluginCfg.Path, err)
		}
		logger.Infof("Plugin '%s' processed successfully.", pluginCfg.Name)
	}

	logger.Info("All plugins processed. Final configuration ready.")
	return &cfg, nil
}
