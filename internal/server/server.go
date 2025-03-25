package server

import (
	"fmt"
	"net/http"
	"time"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus" //to fix duplicate log "/" print we would need to import mux
)

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Build the address
	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"ruta":       "/",
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Solicitud recibida")
		fmt.Fprintln(w, "¡Funciona! :D, esta vez en conjunto")
	})

	http.HandleFunc("/david", func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"ruta":       "/david",
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Solicitud recibida")
		fmt.Fprintf(w, "Hola, David! 🕐 %s\n", time.Now().Format("15:04:05"))
	})

	http.HandleFunc("/nacho", func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"ruta":       "/nacho",
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Solicitud recibida")
		fmt.Fprintf(w, "Hola, Nacho! 🕐 %s\n", time.Now().Format("15:04:05"))
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"ruta":       "/healthz",
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Solicitud recibida")

		// Assuming you have a plugin that provides health check parameters
		plugin, err := pluginManager.GetPlugin("health-check-plugin")
		if err != nil {
			logger.Errorf("Error obteniendo plugin: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		result, err := plugin.Execute(nil) // Pass any required parameters here
		if err != nil {
			logger.Errorf("Error ejecutando plugin: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Health Check Result: %s\n", result)
	})
	http.HandleFunc("/flow", handleFlow(cfg, logger))

	// Start server
	logger.Infof("Escuchando en http://%s", address)
	server := &http.Server{
		Addr:    address,
		Handler: nil,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func handleFlow(cfg *v1alpha1.Config, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.WithFields(logrus.Fields{
			"ruta":       "/flow",
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Solicitud recibida")
		logger.Infof("Ejecutando flujo de incidentes")
		for _, flow := range cfg.Flows {
			if flow.Name == "incident-flow" {
				executeFlow(flow, logger)
			}
		}
	}
}

func executeFlow(flow v1alpha1.Flow, logger *logrus.Logger) {
	for _, step := range flow.Pipeline {
		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Error obteniendo plugin: %v", err)
			continue
		}

		_, err = plugin.Execute(step.Parameters)
		if err != nil {
			logger.Errorf("Error ejecutando plugin: %v", err)
		} else {
			logger.Infof("Plugin %s ejecutado correctamente", step.PluginRef)
		}
	}
}
