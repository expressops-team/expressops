// Package config provides functionality for loading and managing configuration files
package config

import (
	"context"
	"fmt"
	"os"
	"reflect" // used for setting default values via struct tags
	"strconv"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// InitializeLogger creates a basic logger with default configuration
func InitializeLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout

	// Use basic text formatter initially
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}

// LoadConfig loads the configuration from the specified YAML file
// Load the configuration from YAML

func LoadConfig(ctx context.Context, path string, logger *logrus.Logger) (*v1alpha1.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables in the config file
	expandedData := os.ExpandEnv(string(data))

	// Unmarshal YAML data into Config struct
	var cfg v1alpha1.Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	// Apply defaults from struct tags
	applyDefaults(&cfg, logger)

	// Override with environment variables if they exist
	ApplyEnvironmentOverrides(&cfg, logger)

	logger.Info("Base configuration loaded. Processing plugins...")

	// Process each plugin in the configuration
	for i := range cfg.Plugins {
		pluginCfg := &cfg.Plugins[i]

		// Skip if plugin name is empty (commented out in config)
		if pluginCfg.Name == "" {
			logger.Debug("Skipping commented out plugin entry")
			continue
		}

		logger.Debugf("Loading plugin code: %s (Path: %s)", pluginCfg.Name, pluginCfg.Path)
		if err := pluginManager.LoadPlugin(ctx, pluginCfg.Path, pluginCfg.Name, pluginCfg.Config, logger); err != nil {
			// Detailed error message
			return nil, fmt.Errorf("error loading plugin '%s' from '%s': %w\n"+
				"Please check:\n"+
				"- The plugin file exists\n"+
				"- The plugin was built for the correct architecture\n"+
				"- You have the necessary permissions to access the file",
				pluginCfg.Name, pluginCfg.Path, err)
		}
		logger.Infof("Plugin '%s' processed successfully.", pluginCfg.Name)
	}

	logger.Info("All plugins processed. Final configuration ready.")
	return &cfg, nil
}

// applyDefaults applies default values from struct tags if not set
func applyDefaults(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Apply server defaults if not set
	serverType := reflect.TypeOf(cfg.Server)
	serverValue := reflect.ValueOf(&cfg.Server).Elem()

	for i := 0; i < serverType.NumField(); i++ {
		field := serverType.Field(i)
		defaultValue := field.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

		fieldValue := serverValue.Field(i)

		// Only apply default if field is zero value
		if fieldValue.Kind() == reflect.Int && fieldValue.Int() == 0 {
			if intValue, err := strconv.Atoi(defaultValue); err == nil {
				fieldValue.SetInt(int64(intValue))
				logger.Debugf("Applied default value for %s: %s", field.Name, defaultValue)
			}
		} else if fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			fieldValue.SetString(defaultValue)
			logger.Debugf("Applied default value for %s: %s", field.Name, defaultValue)
		}
	}
}

// ApplyEnvironmentOverrides overrides configuration with environment variables
func ApplyEnvironmentOverrides(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Server configuration
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = port
			logger.Infof("SERVER_PORT set from environment variable: %d", port)
		} else {
			logger.Warnf("Invalid SERVER_PORT value: %s", portStr)
		}
	}

	if address := os.Getenv("SERVER_ADDRESS"); address != "" {
		cfg.Server.Address = address
		logger.Infof("SERVER_ADDRESS set from environment variable: %s", address)
	}

	if timeoutStr := os.Getenv("TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			cfg.Server.TimeoutSec = timeout
			logger.Infof("TIMEOUT_SECONDS set from environment variable: %d", timeout)
		} else {
			logger.Warnf("Invalid TIMEOUT_SECONDS value: %s", timeoutStr)
		}
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
		logger.Infof("LOG_LEVEL set from environment variable: %s", level)
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
		logger.Infof("LOG_FORMAT set from environment variable: %s", format)
	}
}

// ConfigureLogger sets up the logger based on the provided configuration

func ConfigureLogger(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Configure based on config
	var formatter logrus.Formatter
	switch cfg.Logging.Format {
	case "json":
		formatter = &logrus.JSONFormatter{}
	default: // plain text
		formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		}
	}
	logger.SetFormatter(formatter)

	logLevel := logrus.DebugLevel // default level

	if cfg.Logging.Level != "" {
		if level, err := logrus.ParseLevel(cfg.Logging.Level); err == nil {
			logLevel = level
		}
	}

	logger.SetLevel(logLevel)
	logger.Infof("Logger configured with format=%s and level=%s", cfg.Logging.Format, logLevel)
}
