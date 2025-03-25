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
	var cfg v1alpha1.Config
	// with the yaml package("gopkg.in/yaml.v3"), we unmarshal the data into the cfg variable
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// for each plugin in the config, we load the plugin, get the plugin instance and initialize it
	for _, p := range cfg.Plugins {
		if err := pluginManager.LoadPlugin(ctx, p.Path, p.Name, p.Config, logger); err != nil {
			return nil, fmt.Errorf("error cargando plugin %s: %v", p.Name, err)
		}
	}

	return &cfg, nil
}
