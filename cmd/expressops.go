package main

import (
	"context"
	"expressops/internal/config" // imports the internal/config package
	"expressops/internal/server" // imports the server package
	"flag"
	//logger
)

func main() {
	// Initialize basic logger
	logger := config.InitializeLogger()

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

	// 2º configure logger based on loaded config
	config.ConfigureLogger(cfg, logger)

	// 3º start the server
	server.StartServer(cfg, logger)
}
