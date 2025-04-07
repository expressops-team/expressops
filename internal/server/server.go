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
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flow registered: %s", flow.Name)
	}
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	initializeFlowRegistry(cfg, logger)

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Health check request received")

		// You could add actual health checks here
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("OK"))
	})

	timeout := time.Duration(cfg.Server.TimeoutSec) * time.Second

	// ONLY one generic handler that will handle all flows
	http.HandleFunc("/flow", dynamicFlowHandler(logger, timeout))
	logger.Infof("Server listening on http://%s", address)

	// help for the user
	logger.Infof("➡️ curl http://%s/flow?flowName=<flow_name> ⬅️", address)

	srv := &http.Server{Addr: address}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error starting server: %v", err)
	}
}

// dynamicFlowHandler handles requests to /flow and executes configured flows
func dynamicFlowHandler(logger *logrus.Logger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout) // if it takes more than 4 seconds, it will be killed

		defer cancel()

		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			http.Error(w, "Must indicate flowName", http.StatusBadRequest)
			return
		}

		flow, exists := flowRegistry[flowName]
		if !exists {
			http.Error(w, fmt.Sprintf("Flow '%s' not found", flowName), http.StatusNotFound)
			return
		}

		logger.WithFields(logrus.Fields{
			"flow":       flowName,
			"ip":         r.RemoteAddr,
			"user_agent": r.UserAgent(),
		}).Info("Executing requested flow dynamically")

		paramsRaw := r.URL.Query().Get("params")
		additionalParams := parseParams(paramsRaw)

		results := executeFlow(ctx, flow, additionalParams, r, logger)

		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"flow":    flowName,
			"success": true,
			"count":   len(results),
		}

		for _, res := range results {
			if result, ok := res.(map[string]interface{}); ok {
				if _, hasError := result["error"]; hasError {
					response["success"] = false
					break
				}
			}
		}

		json.NewEncoder(w).Encode(response)
	}
}

// transform the params string to a map[string]interface{}
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

// step by step execution of the flow
func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, r *http.Request, logger *logrus.Logger) []interface{} {
	var results []interface{}
	shared := make(map[string]interface{})

	for k, v := range additionalParams {
		shared[k] = v //necessary for the new parameter "shared"
	}

	var lastResult interface{}
	for _, step := range flow.Pipeline {
		// Skip commented plugins
		if step.PluginRef == "" {
			continue
		}

		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Plugin not found: %s - %v", step.PluginRef, err)
			results = append(results, map[string]interface{}{
				"plugin": step.PluginRef,
				"error":  fmt.Sprintf("Plugin not found: %v", err),
			})
			continue
		}

		for k, v := range step.Parameters {
			shared[k] = v
		}

		shared["previous_result"] = lastResult

		shared["_input"] = lastResult

		logger.Infof("Executing plugin: %s", step.PluginRef)
		res, err := plugin.Execute(ctx, r, &shared)
		if err != nil {
			logger.Errorf("Error executing plugin: %s - %v", step.PluginRef, err)
			results = append(results, map[string]interface{}{
				"plugin": step.PluginRef,
				"error":  fmt.Sprintf("Error: %v", err),
			})
			continue
		}

		var formattedResult string
		if res != nil {
			var fmtErr error
			formattedResult, fmtErr = plugin.FormatResult(res)
			if fmtErr != nil {
				logger.Warnf("Error formatting result from %s: %v", step.PluginRef, fmtErr)
				formattedResult = fmt.Sprintf("%v", res)
			}
		}

		// Add result to results array
		result := map[string]interface{}{
			"plugin": step.PluginRef,
			"result": res,
		}

		// Only add formatted_result if it exists
		if formattedResult != "" {
			result["formatted_result"] = formattedResult
		}

		results = append(results, result)
		if step.PluginRef == "formatter-plugin" {
			logger.Infof("Result from %s: [long output, check the slack channel ;D]", step.PluginRef)
		} else {
			if len(formattedResult) > 100 {
				logger.Infof("Result from %s: %s...", step.PluginRef, formattedResult[:100])
			} else {
				logger.Infof("Result from %s: %s", step.PluginRef, formattedResult)
			}
		}

		lastResult = res
	}

	return results
}
