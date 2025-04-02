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

	// Expand environment variables in the config file
	expandedData := os.ExpandEnv(string(data))

	// declares a variable of type Config
	// with the yaml package("gopkg.in/yaml.v3"), we unmarshal the data into the cfg variable
	var cfg v1alpha1.Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	logger.Info("Base configuration loaded. Processing plugins...")

	// Process each plugin in the configuration
	for i := range cfg.Plugins {
		pluginCfg := &cfg.Plugins[i]

		logger.Debugf("Loading plugin code: %s (Path: %s)", pluginCfg.Name, pluginCfg.Path)
		if err := pluginManager.LoadPlugin(ctx, pluginCfg.Path, pluginCfg.Name, pluginCfg.Config, logger); err != nil {
			return nil, fmt.Errorf("error loading plugin '%s' from '%s': %w", pluginCfg.Name, pluginCfg.Path, err)
		}
		logger.Infof("Plugin '%s' processed successfully.", pluginCfg.Name)
	}

	logger.Info("All plugins processed. Final configuration ready.")
	return &cfg, nil
}
