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
	// This ensures all metrics are registered with the Prometheus registry
}

func main() {
	// Initialize basic logger
	logger := config.InitializeLogger() // Mantén tu logger inicial para config y errores tempranos

	ctx := context.Background() // crea un nuevo contexto

	// Inicializar OpenTelemetry TracerProvider
	tp, err := tracing.InitTracerProvider("expressops-service") // Define el nombre de tu servicio
	if err != nil {
		logger.Fatalf("Failed to initialize tracer provider: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Printf("Error shutting down tracer provider: %v", err)
		}
	}() // Asegura que se llame a Shutdown

	// Obtener el tracer
	tracer := tracing.GetTracer()

	// Iniciar un span para la ejecución principal de main
	// El contexto (ctx) original se pasa al span, y tracer.Start devuelve un nuevo contexto (ctxMain) con el span activo.
	ctxMain, mainSpan := tracer.Start(ctx, "main-execution")
	defer mainSpan.End() // Asegura que el span principal se cierre al final de main

	// Parse the command line flags to get the config file path
	var configPath string
	flag.StringVar(&configPath, "config", "docs/samples/config.yaml", "Path to YAML configuration file")
	flag.Parse()

	// 1º load the config from YAML
	// Usa ctxMain aquí si quieres que la carga de config sea parte del span "main-execution"
	cfg, err := config.LoadConfig(ctxMain, configPath, logger)
	if err != nil {
		mainSpan.RecordError(err) // Registra el error en el span
		// mainSpan.SetStatus(codes.Error, err.Error()) // También puedes establecer el estado del span como error
		logger.Fatalf("Error loading configuration: %v", err)
	}

	// 2º configure logger based on loaded config
	config.ConfigureLogger(cfg, logger) // Reconfigura el logger con la configuración cargada si es necesario

	// 3º start the server
	// Si StartServer toma un contexto, pasa ctxMain
	server.StartServer(cfg, logger) // Eliminado ctx para que coincida con la firma actual
}
