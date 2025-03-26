// internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"expressops/api/v1alpha1"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// registry of flows
var flowRegistry map[string]v1alpha1.Flow

func initializeFlowRegistry(cfg *v1alpha1.Config, logger *logrus.Logger) {
	flowRegistry = make(map[string]v1alpha1.Flow)
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flujo registrado: %s", flow.Name)
	}
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Configure logger to include timestamps
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Initializes the map at server startup
	initializeFlowRegistry(cfg, logger)

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	// Basic route to verify that the server is up
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Solicitud en ruta raíz recibida")
		fmt.Fprintf(w, "ExpressOps activo 🟢 \n")
	})

	// plugins registered dynamically
	for _, pluginConf := range cfg.Plugins {
		pluginName := pluginConf.Name
		route := "/flows/" + pluginName

		http.HandleFunc(route, func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
				defer cancel()

				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				plugin, err := pluginManager.GetPlugin(name)
				if err != nil {
					http.Error(w, "Plugin no encontrado", http.StatusNotFound)
					return
				}

				result, err := plugin.Execute(ctx, nil)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if formatter, ok := plugin.(interface {
					FormatResult(interface{}) (string, error)
				}); ok {
					formatted, err := formatter.FormatResult(result)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					fmt.Fprint(w, formatted)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
			}
		}(pluginName))
	}

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

		// obtain flowName from query parameter "flowName"
		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			http.Error(w, "Debe indicar flowName", http.StatusBadRequest)
			return
		}

		// check if the flow exists in the flow registry
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

		// read additional parameters (if they exist?)
		paramsRaw := r.URL.Query().Get("params")
		additionalParams := parseParams(paramsRaw)

		// execute the found flow
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

	// split the params by ;
	pairs := strings.Split(paramsRaw, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
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

		// combine the parameters from the YAML with the additional parameters
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
