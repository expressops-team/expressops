package config

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	// We import the LoadPlugin and GetPlugin functions from the pluginManager package
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

	// Sobrescribir con variables de entorno, si existen
	ApplyEnvironmentOverrides(&cfg, logger)

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

// ApplyEnvironmentOverrides sobrescribe la configuración con variables de entorno
func ApplyEnvironmentOverrides(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Configuración del servidor
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = port
			logger.Infof("SERVER_PORT configurado desde la variable de entorno: %d", port)
		} else {
			logger.Warnf("Variable SERVER_PORT inválida: %s", portStr)
		}
	}

	if address := os.Getenv("SERVER_ADDRESS"); address != "" {
		cfg.Server.Address = address
		logger.Infof("SERVER_ADDRESS configurado desde la variable de entorno: %s", address)
	}

	if timeoutStr := os.Getenv("TIMEOUT_SECONDS"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			cfg.Server.TimeoutSec = timeout
			logger.Infof("TIMEOUT_SECONDS configurado desde la variable de entorno: %d", timeout)
		} else {
			logger.Warnf("Variable TIMEOUT_SECONDS inválida: %s", timeoutStr)
		}
	}

	// Configuración de logging
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
		logger.Infof("LOG_LEVEL configurado desde la variable de entorno: %s", level)
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
		logger.Infof("LOG_FORMAT configurado desde la variable de entorno: %s", format)
	}
}

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
	logger.Infof("Logger configurado con format=%s y level=%s", cfg.Logging.Format, logLevel)
}
