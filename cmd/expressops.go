package main

import (
	"context"
	"expressops/internal/config" // imports the internal/config package
	"expressops/internal/server" // imports the server package
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus" //logger
)

func main() {
	// Initialize logrus logger
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	ctx := context.Background() // creates a new context para gestionar timeouts, cancelaciones, etc.

	// builds the path to the config file
	configPath := filepath.Join("docs", "samples", "config.yaml")

	// 1º load the config from YAML
	cfg, err := config.LoadConfig(ctx, configPath, logger)
	if err != nil {
		logger.Fatalf("Error al cargar la configuración: %v", err)
	}

	// 2º start the server
	server.StartServer(cfg, logger)
}
