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
	fmt.Print("\n") // whitespace
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flujo registrado: %s", flow.Name)
	}
	fmt.Print("\n") // whitespace
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
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

				result, err := plugin.Execute(ctx, r, nil)
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
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second) // if it takes more than 4 seconds, it will be killed
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

		//

		results := executeFlow(ctx, flow, additionalParams, logger, r)

		w.Header().Set("Content-Type", "application/json")

		fmt.Fprintf(w, "Flujo '%s' ejecutado correctamente.\n", flowName)
		for _, result := range results {
			if resultMap, ok := result.(map[string]interface{}); ok {
				pluginName := resultMap["plugin"].(string)
				plugin, err := pluginManager.GetPlugin(pluginName)
				if err != nil {
					fmt.Fprintf(w, "Plugin: %s\nError: %v\n\n", pluginName, err)
					continue
				}

				// if plugin implements formatter --> use it
				if formatter, ok := plugin.(interface {
					FormatResult(interface{}) (string, error)
				}); ok {
					formatted, err := formatter.FormatResult(resultMap["resultado"])
					if err != nil {
						fmt.Fprintf(w, "Plugin: %s\nError formateando resultado: %v\n\n", pluginName, err)
						continue
					}
					fmt.Fprintf(w, "Plugin: %s\n%s\n\n", pluginName, formatted)
				} else {
					// If no formatter, raw result
					fmt.Fprintf(w, "Plugin: %s\nResultado: %v\n\n", pluginName, resultMap["resultado"])
				}
			}
		}
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

func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, logger *logrus.Logger, r *http.Request) []interface{} {
	var results []interface{}
	var lastResult interface{} = nil // <-stores the previous result

	// shared whiteboard
	// Created ONLY ONCE at the start of the flow.

	sharedData := make(map[string]any) // <- shared data between plugins

	//You can pre-fill it with something if you want, for example, URL parameters:
	for k, v := range additionalParams {
		sharedData[k] = v
	}

	// iterate over the steps in the flow
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

		//Add the previuos result to SharedData(replaces '_input')
		if lastResult != nil {
			sharedData["previous_result"] = lastResult
		}

		// Add YAML parameters to sharedData (optional)
		for k, v := range step.Parameters {

			if _, exists := sharedData[k]; !exists {
				sharedData[k] = v
			} else {
				logger.Warnf("La clave '%s' de los parámetros del step ya existe en sharedData, no se sobrescribirá.", k)
			}
		}

		// ***** PASO 2: Ejecutar el plugin pasando el request y la pizarra compartida *****
		// Pasamos 'r' (el http.Request original)
		// Pasamos '&sharedData' (la dirección/puntero a nuestra pizarra compartida)

		// El plugin puede modificar la pizarra compartida y devolver un resultado

		res, err := plugin.Execute(ctx, r, &sharedData) //<-- ¡LA NUEVA LLAMADA!
		if err != nil {
			logger.Errorf("Error ejecutando plugin %s: %v", step.PluginRef, err)
			lastResult = nil

		} else {
			results = append(results, map[string]interface{}{"plugin": step.PluginRef, "resultado": res})
			lastResult = res // <- guardar resultado para el siguiente paso
		}
	}

	return results
}
