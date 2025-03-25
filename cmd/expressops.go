package main

import (
	"expressops/internal/config" // imports the internal/config package
	"expressops/internal/server" // imports the server package
	"fmt"
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

	// builds the path to the config file
	configPath := filepath.Join("docs", "samples", "config.yaml")

	// 1º load the config from YAML
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatalf("Error al cargar la configuración: %v", err)
	}

	// prints the config loaded from the YAML file
	fmt.Printf("Configuración: %+v\n", cfg)
	fmt.Printf("Nivel de logging: %s\n", cfg.Logging.Level)
	fmt.Printf("Formato de logging: %s\n", cfg.Logging.Format)

	// 2º start the server
	server.StartServer(cfg, logger)
}
