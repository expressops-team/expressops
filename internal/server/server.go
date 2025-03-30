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
	fmt.Print("\n")
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flujo registrado: %s", flow.Name)
	}
	fmt.Print("\n")
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

				result, err := plugin.Execute(ctx, r, &map[string]any{})
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

	fmt.Println("\n\033[31mTemplate para flujos:\033[0m")
	fmt.Printf("\n\033[37m ➡️ \033[0m \033[32mcurl http://%s/flow?flowName=<nombre_del_flujo>\033[0m \033[37m ⬅️ \033[0m\n\n", address)

	server := &http.Server{
		Addr: address,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error al iniciar servidor: %v", err)
	}
}

// dynamicFlowHandler handles requests to /flow and executes configured flows
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

				// if plugin implements formatter
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

// parseParams transforms a string of type "key:value;key2:value2" into a map[string]interface{}
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

// executeFlow executes the steps in the flow sequentially and returns the results

func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, logger *logrus.Logger, r *http.Request) []interface{} {
	var results []interface{}
	var lastResult interface{} = nil // <-stores the previous result

	// shared whiteboard
	shared := &map[string]any{} // <- shared data between plugins

	// iterate over the steps in the flow
	for _, step := range flow.Pipeline {
		logger.Infof("Ejecutando plugin: %s", step.PluginRef)

		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Plugin no encontrado: %v", err)
			results = append(results, map[string]string{"error": err.Error()})
			continue
		}

		// Add the parameters to shaed context

		for k, v := range additionalParams {
			(*shared)[k] = v
		}

		//Add the previuos result to Shared(replaces '_input')
		if lastResult != nil {
			(*shared)["previous_result"] = lastResult
			(*shared)["_input"] = lastResult
		}

		// Add YAML parameters to sharedData (optional)
		for k, v := range step.Parameters {

			if _, exists := (*shared)[k]; !exists {
				(*shared)[k] = v
			} else {
				logger.Warnf("La clave '%s' de los parámetros del step ya existe en shared, no se sobrescribirá.", k)
			}
		}

		// ***** STEP 2: Run the plugin by passing the request and the shared whiteboard *****
		// We pass 'r' (the original http.Request)
		// We pass '&sharedData' (the address/pointer to our shared whiteboard)

		// The plugin can modify the shared whiteboard and return a result

		res, err := plugin.Execute(ctx, r, shared) //<-- execute the plugin
		if err != nil {
			logger.Errorf("Error ejecutando plugin %s: %v", step.PluginRef, err)
			lastResult = nil

		} else {
			results = append(results, map[string]interface{}{"plugin": step.PluginRef, "resultado": res})
			lastResult = res // <- save result for the next step
		}
	}

	return results
}
