package main

import (
	"context"
	"expressops/internal/config" // imports the internal/config package
	"expressops/internal/metrics"
	"expressops/internal/server"  // imports the server package
	"expressops/internal/tracing" // Import the tracing package
	"flag"
	"runtime"
	//logger
)

// Initialize metrics when package is loaded
func init() {
	// Register and initialize basic metrics
	metrics.SetActivePlugins(0)

	// Initialize resource metrics with initial values
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.RecordMemoryUsage(float64(m.Alloc))
	metrics.RecordCpuUsage(0.0)
	metrics.UpdateStorageUsage(0.0)
	metrics.UpdateConcurrentPlugins(0)

	// Log initialization
}

func main() {
	// Initialize basic logger
	logger := config.InitializeLogger() // Mantén tu logger inicial para config y errores tempranos

	ctx := context.Background() // creates a new context to manage timeouts, cancelaciones, etc.

	// Initialize OpenTelemetry TracerProvider
	tp, err := tracing.InitTracerProvider("expressops-service") // Define the name of your service
	if err != nil {
		logger.Fatalf("Failed to initialize tracer provider: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Printf("Error shutting down tracer provider: %v", err)
		}
	}() // Ensure Shutdown

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
