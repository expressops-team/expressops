package server

import (
	"context"
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

		plugin, err := pluginManager.GetPlugin("health-check-plugin")
		if err != nil {
			logger.Errorf("Error obteniendo plugin: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		resultRaw, err := plugin.Execute(r.Context(), nil)
		if err != nil {
			logger.Errorf("Error ejecutando plugin: %v", err)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		resultMap, ok := resultRaw.(map[string]interface{})
		if !ok {
			logger.Errorf("El resultado no es del tipo esperado")
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
			return
		}

		// Imprime información formateada
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		// CPU
		fmt.Fprintf(w, "\n🔹 CPU usage:\n")
		if cpu, ok := resultMap["cpu"].(map[string]interface{}); ok {
			fmt.Fprintf(w, "  Usage: %.2f%%\n", cpu["usage_percent"].(float64))
		}

		// Memory
		fmt.Fprintf(w, "\n🔹 Memory usage:\n")
		if memory, ok := resultMap["memory"].(map[string]interface{}); ok {
			fmt.Fprintf(w, "  Total: %.2f GB\n", float64(memory["total"].(uint64))/1e9)
			fmt.Fprintf(w, "  Used: %.2f GB\n", float64(memory["used"].(uint64))/1e9)
			fmt.Fprintf(w, "  Free: %.2f GB\n", float64(memory["free"].(uint64))/1e9)
			fmt.Fprintf(w, "  Usage: %.2f%%\n", memory["used_percent"].(float64))
		}

		// Disk
		fmt.Fprintf(w, "\n🔹 Disk usage:\n")
		if disk, ok := resultMap["disk"].(map[string]interface{}); ok {
			for mount, usageRaw := range disk {
				usage := usageRaw.(map[string]interface{})
				fmt.Fprintf(w, "  📁 Mount: %s\n", mount)
				fmt.Fprintf(w, "    Total: %.2f GB\n", float64(usage["total"].(uint64))/1e9)
				fmt.Fprintf(w, "    Used: %.2f GB\n", float64(usage["used"].(uint64))/1e9)
				fmt.Fprintf(w, "    Free: %.2f GB\n", float64(usage["free"].(uint64))/1e9)
				fmt.Fprintf(w, "    Usage: %.2f%%\n\n", usage["used_percent"].(float64))
			}
		}

		// Estado general
		fmt.Fprintf(w, "\n🔹 Health Checks:\n")
		if status, ok := resultMap["health_status"].(map[string]interface{}); ok {
			for k, v := range status {
				fmt.Fprintf(w, "  %s: %s\n", k, v.(string))
			}
		}
	})

	http.HandleFunc("/test-context", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) // 5 segundos máximo
		defer cancel()

		plugin, err := pluginManager.GetPlugin("sleep-plugin")
		if err != nil {
			http.Error(w, "Plugin no encontrado", http.StatusInternalServerError)
			return
		}

		resultado, err := plugin.Execute(ctx, nil)
		if err != nil {
			fmt.Fprintf(w, "El plugin fue cancelado: %v\n", err)
			return
		}

		fmt.Fprintf(w, "Resultado: %v\n", resultado)
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
				executeFlow(r.Context(), flow, logger)
			}
		}
	}
}

func executeFlow(ctx context.Context, flow v1alpha1.Flow, logger *logrus.Logger) {
	for _, step := range flow.Pipeline {
		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Error obteniendo plugin: %v", err)
			continue
		}

		_, err = plugin.Execute(ctx, step.Parameters)
		if err != nil {
			logger.Errorf("Error ejecutando plugin: %v", err)
		} else {
			logger.Infof("Plugin %s ejecutado correctamente", step.PluginRef)
		}
	}
}
