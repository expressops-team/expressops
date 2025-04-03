package main

import (
	"context"
	"expressops/internal/config" // imports the internal/config package
	"expressops/internal/server" // imports the server package
	"flag"
	"os"

	"github.com/sirupsen/logrus" //logger
)

func main() {
	// Initialize logrus logger
	logger := logrus.New()
	logger.Out = os.Stdout

	ctx := context.Background() // creates a new context to manage timeouts, cancelaciones, etc.

	// Parse the command line flags to get the config file path
	var configPath string
	flag.StringVar(&configPath, "config", "docs/samples/config.yaml", "Path to YAML configuration file")
	flag.Parse()

	// 1º load the config from YAML
	cfg, err := config.LoadConfig(ctx, configPath, logger)
	if err != nil {
		logger.Fatalf("Error loading configuration: %v", err)
	}

	var formatter logrus.Formatter
	switch cfg.Logging.Format {
	case "json":
		formatter = &logrus.JSONFormatter{}
	case "verbose":
		formatter = &logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		}
	default: // fallback to plain text
		formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		}
	}
	logger.SetFormatter(formatter)

	logger.SetLevel(logrus.DebugLevel) // set the log level to debug

	// 2º start the server
	server.StartServer(cfg, logger)
}
