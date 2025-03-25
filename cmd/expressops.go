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
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	ctx := context.Background() // creates a new context to manage timeouts, cancelaciones, etc.

	// Parse the command line flags to get the config file path
	var configPath string
	flag.StringVar(&configPath, "config", "docs/samples/config.yaml", "Ruta al archivo YAML de configuración")
	flag.Parse()

	// 1º load the config from YAML
	cfg, err := config.LoadConfig(ctx, configPath, logger)
	if err != nil {
		logger.Fatalf("Error al cargar la configuración: %v", err)
	}

	// 2º start the server
	server.StartServer(cfg, logger)
}
