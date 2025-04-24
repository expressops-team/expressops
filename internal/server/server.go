// internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
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

		// Only check if this is the all-flows flow
		isAllFlowsFlow := flowName == "all-flows"
		if isAllFlowsFlow {
			logger.Info("all-flows detected - showing complete output")
		}

		results := executeFlow(ctx, flow, additionalParams, r, logger, isAllFlowsFlow)

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
func executeFlow(ctx context.Context, flow v1alpha1.Flow, additionalParams map[string]interface{}, r *http.Request, logger *logrus.Logger, isAllFlowsFlow bool) []interface{} {
	var results []interface{}
	shared := make(map[string]interface{})

	for k, v := range additionalParams {
		shared[k] = v //necessary for the "new" parameter "shared"
	}

	// Add flow registry to shared context
	shared["flow_registry"] = flowRegistry

	var lastResult interface{}
	var i int = 0

	for i < len(flow.Pipeline) {
		currentStep := flow.Pipeline[i]
		i++

		// Skip commented plugins 			==> WILL BE REMOVED IN THE FUTURE <==
		if currentStep.PluginRef == "" {
			continue
		}

		// Check if we need to run this step in parallel with the next ones
		var parallelSteps []v1alpha1.Step
		parallelSteps = append(parallelSteps, currentStep)

		// Collect consecutive parallel steps
		for i < len(flow.Pipeline) && flow.Pipeline[i].Parallel {
			if flow.Pipeline[i].PluginRef != "" { // Skip commented plugins
				parallelSteps = append(parallelSteps, flow.Pipeline[i])
			}
			i++
		}

		// If we have multiple steps to execute in parallel
		if len(parallelSteps) > 1 {
			logger.Infof("Running %d plugins in parallel", len(parallelSteps))

			var wg sync.WaitGroup
			var mu sync.Mutex
			parallelResults := make([]interface{}, len(parallelSteps))

			parallelShared := make([]map[string]interface{}, len(parallelSteps))
			for j := range parallelSteps {
				parallelShared[j] = make(map[string]interface{})
				for k, v := range shared {
					parallelShared[j][k] = v
				}

				// Add step parameters to shared context
				for k, v := range parallelSteps[j].Parameters {
					parallelShared[j][k] = v
				}

				parallelShared[j]["previous_result"] = lastResult
				parallelShared[j]["_input"] = lastResult
			}

			// Execute each plugin in parallel
			for j, step := range parallelSteps {
				wg.Add(1)
				go func(idx int, s v1alpha1.Step, pShared map[string]interface{}) {
					defer wg.Done()

					plugin, err := pluginManager.GetPlugin(s.PluginRef)
					if err != nil {
						logger.Errorf("Plugin not found: %s - %v", s.PluginRef, err)
						mu.Lock()
						parallelResults[idx] = map[string]interface{}{
							"plugin": s.PluginRef,
							"error":  fmt.Sprintf("Plugin not found: %v", err),
						}
						mu.Unlock()
						return
					}

					logger.Infof("Executing plugin (parallel): %s", s.PluginRef)
					res, err := plugin.Execute(ctx, r, &pShared)
					if err != nil {
						logger.Errorf("Error executing plugin: %s - %v", s.PluginRef, err)
						mu.Lock()
						parallelResults[idx] = map[string]interface{}{
							"plugin": s.PluginRef,
							"error":  fmt.Sprintf("Error: %v", err),
						}
						mu.Unlock()
						return
					}

					var formattedResult string
					if res != nil {
						var fmtErr error
						formattedResult, fmtErr = plugin.FormatResult(res)
						if fmtErr != nil {
							logger.Warnf("Error formatting result from %s: %v", s.PluginRef, fmtErr)
							formattedResult = fmt.Sprintf("%v", res)
						}
					}

					mu.Lock()
					result := map[string]interface{}{
						"plugin": s.PluginRef,
						"result": res,
					}

					if formattedResult != "" {
						result["formatted_result"] = formattedResult
					}

					parallelResults[idx] = result
					mu.Unlock()

					// Log the result
					if s.PluginRef == "formatter-plugin" {
						logger.Infof("Result from %s (parallel): [long output, check the slack channel]", s.PluginRef)
					} else if strings.HasPrefix(formattedResult, "__MULTILINE_LOG__") {
						logLines := strings.Split(formattedResult, "__MULTILINE_LOG__")
						logger.Infof("Result from %s (parallel, multi-line output):", s.PluginRef)
						for _, line := range logLines {
							if line == "" {
								continue
							}
							logger.Info(line)
						}
					} else {
						if !isAllFlowsFlow && len(formattedResult) > 100 {
							logger.Infof("Result from %s (parallel): %s...", s.PluginRef, formattedResult[:100])
						} else {
							logger.Infof("Result from %s (parallel): %s", s.PluginRef, formattedResult)
						}
					}
				}(j, step, parallelShared[j])
			}

			// Wait for all parallel steps to complete
			wg.Wait()

			for _, result := range parallelResults {
				if result != nil {
					results = append(results, result)

					// Get the last non-error result to use as input for next plugin
					if res, ok := result.(map[string]interface{}); ok {
						if _, hasError := res["error"]; !hasError {
							if pluginResult, exists := res["result"]; exists {
								lastResult = pluginResult
							}
						}
					}
				}
			}
		} else {
			// if run 1 plugin normally (no parallelization)
			plugin, err := pluginManager.GetPlugin(currentStep.PluginRef)
			if err != nil {
				logger.Errorf("Plugin not found: %s - %v", currentStep.PluginRef, err)
				results = append(results, map[string]interface{}{
					"plugin": currentStep.PluginRef,
					"error":  fmt.Sprintf("Plugin not found: %v", err),
				})
				continue
			}

			for k, v := range currentStep.Parameters {
				shared[k] = v
			}

			shared["previous_result"] = lastResult
			shared["_input"] = lastResult

			logger.Infof("Executing plugin: %s", currentStep.PluginRef)
			res, err := plugin.Execute(ctx, r, &shared)
			if err != nil {
				logger.Errorf("Error executing plugin: %s - %v", currentStep.PluginRef, err)
				results = append(results, map[string]interface{}{
					"plugin": currentStep.PluginRef,
					"error":  fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			var formattedResult string
			if res != nil {
				var fmtErr error
				formattedResult, fmtErr = plugin.FormatResult(res)
				if fmtErr != nil {
					logger.Warnf("Error formatting result from %s: %v", currentStep.PluginRef, fmtErr)
					formattedResult = fmt.Sprintf("%v", res)
				}
			}

			result := map[string]interface{}{
				"plugin": currentStep.PluginRef,
				"result": res,
			}

			// Only add formatted_result if it exists
			if formattedResult != "" {
				result["formatted_result"] = formattedResult
			}

			results = append(results, result)
			if currentStep.PluginRef == "formatter-plugin" {
				logger.Infof("Result from %s: [long output, check the slack channel ;D]", currentStep.PluginRef)
			} else if strings.HasPrefix(formattedResult, "__MULTILINE_LOG__") {
				// Handle multi-line logging
				logLines := strings.Split(formattedResult, "__MULTILINE_LOG__")
				logger.Infof("Result from %s (multi-line output):", currentStep.PluginRef)
				for _, line := range logLines {
					if line == "" {
						continue
					}
					logger.Info(line)
				}
			} else {
				// Show full output for [all-flows] flow, truncate others if they're too long
				if !isAllFlowsFlow && len(formattedResult) > 100 {
					logger.Infof("Result from %s: %s...", currentStep.PluginRef, formattedResult[:100])
				} else {
					logger.Infof("Result from %s: %s", currentStep.PluginRef, formattedResult)
				}
			}

			lastResult = res
		}
	}

	return results
}
