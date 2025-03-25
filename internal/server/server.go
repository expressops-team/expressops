package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus" //to fix duplicate log "/" print we would need to import mux
)

var flowRegistry map[string]v1alpha1.Flow

func initializeFlowRegistry(cfg *v1alpha1.Config, logger *logrus.Logger) {
	flowRegistry = make(map[string]v1alpha1.Flow)
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flujo registrado: %s", flow.Name)
	}
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Initializes the map at server startup
	initializeFlowRegistry(cfg, logger)

	// Dirección del servidor
	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	// Basic route to verify that the server is up
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Solicitud en ruta raíz recibida")
		fmt.Fprintf(w, "ExpressOps activo 🟢")
	})

	// ONLY one generic handler that will handle all flows
	http.HandleFunc("/flow", dynamicFlowHandler(logger))

	logger.Infof("Servidor escuchando en http://%s", address)
	server := &http.Server{
		Addr: address,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error al iniciar servidor: %v", err)
	}
}
func dynamicFlowHandler(logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Obtienes claramente el flujo desde el parámetro query "flowName"
		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			http.Error(w, "Debe indicar flowName", http.StatusBadRequest)
			return
		}

		// Verifica que el flujo exista en el registro de flujos
		flow, exists := flowRegistry[flowName]
		if !exists {
			http.Error(w, fmt.Sprintf("Flujo '%s' no encontrado", flowName), http.StatusNotFound)
			return
		}

		logger.WithFields(logrus.Fields{
			"flujo":      flowName,
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Ejecutando flujo solicitado dinámicamente")

		// Lee claramente los parámetros adicionales (si existen)
		paramsRaw := r.URL.Query().Get("params")
		additionalParams := parseParams(paramsRaw)

		// Ejecuta claramente el flujo encontrado
		results := executeFlow(ctx, flow, additionalParams, logger)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "Flujo '%s' ejecutado correctamente. Resultados: %v", flowName, results)
	}
}

func parseParams(paramsRaw string) map[string]interface{} {
	params := make(map[string]interface{})
	if paramsRaw == "" {
		return params
	}

	pairs := strings.Split(paramsRaw, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

// Create a dynamic handler for each configured flow
func handleFlow(flow v1alpha1.Flow, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		logger.WithFields(logrus.Fields{
			"ruta":       r.URL.Path,
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
			"flujo":      flow.Name,
		}).Info("Ejecutando flujo solicitado")

		results := executeFlow(ctx, flow, map[string]interface{}{}, logger)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "Flujo '%s' ejecutado correctamente. Resultados: %v", flow.Name, results)
	}
}

// Custom Handler for Detailed Health Check
func detailedHealthCheckHandler(flow v1alpha1.Flow, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Solicitud recibida en healthCheckDetailed")

		plugin, err := pluginManager.GetPlugin("health-check-plugin")
		if err != nil {
			logger.Errorf("Error obteniendo plugin: %v", err)
			http.Error(w, "Plugin no encontrado", http.StatusInternalServerError)
			return
		}

		// The result is already a string formatted directly by the plugin
		resultRaw, err := plugin.Execute(r.Context(), nil)
		if err != nil {
			logger.Errorf("Error ejecutando plugin: %v", err)
			http.Error(w, "Error ejecutando plugin", http.StatusInternalServerError)
			return
		}

		resultStr, ok := resultRaw.(string)
		if !ok {
			http.Error(w, "Formato de resultado inesperado", http.StatusInternalServerError)
			return
		}

		// Returns result directly
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, resultStr)
	}
}

// Custom handler to test context timeout
func contextTimeoutTestHandler(flow v1alpha1.Flow, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		logger.Info("Solicitud recibida en contextTimeoutTest")

		plugin, err := pluginManager.GetPlugin("sleep-plugin")
		if err != nil {
			logger.Errorf("Plugin no encontrado: %v", err)
			http.Error(w, "Plugin no encontrado", http.StatusInternalServerError)
			return
		}

		resultado, err := plugin.Execute(ctx, nil)
		if err != nil {
			logger.Warnf("Plugin cancelado o error: %v", err)
			fmt.Fprintf(w, "Plugin cancelado o error: %v\n", err)
			return
		}

		fmt.Fprintf(w, "Plugin ejecutado exitosamente: %v\n", resultado)
	}
}

// Runs the flow and returns results
func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, logger *logrus.Logger) []interface{} {
	var results []interface{}

	for _, step := range flow.Pipeline {
		logger.Infof("Ejecutando plugin: %s", step.PluginRef)

		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Plugin no encontrado: %v", err)
			results = append(results, map[string]string{"error": err.Error()})
			continue
		}

		// combina claramente parámetros del YAML con parámetros adicionales
		params := make(map[string]interface{})
		for k, v := range step.Parameters {
			params[k] = v
		}
		for k, v := range additionalParams {
			params[k] = v
		}

		res, err := plugin.Execute(ctx, params)
		if err != nil {
			logger.Errorf("Error ejecutando plugin %s: %v", step.PluginRef, err)
			results = append(results, map[string]string{"plugin": step.PluginRef, "error": err.Error()})
		} else {
			results = append(results, map[string]interface{}{"plugin": step.PluginRef, "resultado": res})
		}
	}

	return results
}
