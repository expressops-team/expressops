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

// initializeFlowRegistry carga los flujos definidos en el archivo de configuración
func initializeFlowRegistry(cfg *v1alpha1.Config, logger *logrus.Logger) {
	flowRegistry = make(map[string]v1alpha1.Flow)
	fmt.Print("\n") // whitespace
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flujo registrado: %s", flow.Name)
	}
	fmt.Print("\n") // whitespace
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	// Inicializa el registro de flujos
	initializeFlowRegistry(cfg, logger)

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	// Ruta raíz para verificar que el servidor está activo
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Solicitud en ruta raíz recibida")
		fmt.Fprintf(w, "ExpressOps activo 🟢 \n")
	})

	// Rutas dinámicas por cada plugin configurado
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

				result, err := plugin.Execute(ctx, r, &map[string]any{})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(result)
			}
		}(pluginName))
	}

	http.HandleFunc("/flow", dynamicFlowHandler(logger))

	logger.Infof("Servidor escuchando en http://%s", address)

	fmt.Println("\033[31mTemplate para flujos:\033[0m")
	fmt.Printf("\033[37m ➡️ \033[0m \033[32mcurl http://%s/flow?flowName=<nombre_del_flujo>\033[0m \033[37m ⬅️ \033[0m\n\n", address)

	srv := &http.Server{Addr: address}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error al iniciar servidor: %v", err)
	}
}

// dynamicFlowHandler maneja peticiones a /flow y ejecuta flujos configurados
func dynamicFlowHandler(logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()

		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			http.Error(w, "Debe indicar flowName", http.StatusBadRequest)
			return
		}

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

		paramsRaw := r.URL.Query().Get("params")
		additionalParams := parseParams(paramsRaw)

		results := executeFlow(ctx, flow, additionalParams, r, logger)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// parseParams transforma una cadena tipo "key:value;key2:value2" en un map[string]interface{}
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

// executeFlow ejecuta los pasos del flujo secuencialmente y retorna los resultados
func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, r *http.Request, logger *logrus.Logger) []interface{} {
	var results []interface{}
	var lastResult interface{} = nil
	shared := &map[string]any{}

	for _, step := range flow.Pipeline {
		logger.Infof("Ejecutando plugin: %s", step.PluginRef)

		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Plugin no encontrado: %v", err)
			results = append(results, map[string]string{"error": err.Error()})
			continue
		}

		// Add parameters to shared context
		for k, v := range step.Parameters {
			(*shared)[k] = v
		}
		for k, v := range additionalParams {
			(*shared)[k] = v
		}
		if lastResult != nil {
			(*shared)["_input"] = lastResult
		}

		res, err := plugin.Execute(ctx, r, shared)
		if err != nil {
			logger.Errorf("Error ejecutando plugin %s: %v", step.PluginRef, err)
			results = append(results, map[string]string{"plugin": step.PluginRef, "error": err.Error()})
		} else {
			results = append(results, map[string]interface{}{"plugin": step.PluginRef, "resultado": res})
			lastResult = res
		}
	}

	return results
}
