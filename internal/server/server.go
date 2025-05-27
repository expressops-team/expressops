// Package server provides HTTP server functionality for the application
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
	"expressops/internal/metrics"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registry of flows
var flowRegistry map[string]v1alpha1.Flow

// initializeFlowRegistry loads the flows defined in the configuration file
func initializeFlowRegistry(cfg *v1alpha1.Config, logger *logrus.Logger) {
	flowRegistry = make(map[string]v1alpha1.Flow)
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flow registered: %s", flow.Name)
	}
}

// StartServer initializes and starts the HTTP server with the provided configuration
func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	initializeFlowRegistry(cfg, logger)

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		userAgent := r.Header.Get("User-Agent")
		probeTypeLabel := "manual_curl"
		httpStatusCode := http.StatusOK

		// Check if the request is from a Kubernetes liveness/readiness probe
		if strings.HasPrefix(userAgent, "kube-probe/") {
			probeTypeLabel = "kubernetes_probe"
		}

		// Log the request
		logger.Infof("Health check request received on /healthz from User-Agent: %s, identified as: %s", userAgent, probeTypeLabel)

		metrics.IncKubernetesProbe(probeTypeLabel, "/healthz")
		metrics.IncFlowExecuted("healthz", "success")

		// You could add actual health checks here. If they fail, set status to "error" and httpStatusCode accordingly.
		// For now, always success.
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.WithError(err).Error("Error writing response")
		}

		duration := time.Since(startTime).Seconds()
		metrics.ObserveFlowDuration("healthz", "success", duration) // Flow duration for healthz
		metrics.IncHTTPRequestsTotal(r.URL.Path, r.Method, httpStatusCode)
		metrics.ObserveHTTPRequestDuration(r.URL.Path, r.Method, httpStatusCode, duration)
	})

	timeout := time.Duration(cfg.Server.TimeoutSec) * time.Second

	// ONLY one generic handler that will handle all flows
	http.HandleFunc("/flow", dynamicFlowHandler(logger, timeout))
	logger.Infof("Server listening on http://%s", address)

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Metrics endpoint registered at /metrics")

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

		metrics.IncActiveFlowHandlers()
		defer metrics.DecActiveFlowHandlers()

		startTime := time.Now()
		httpStatusCode := http.StatusOK

		ctx, cancel := context.WithTimeout(r.Context(), timeout) // if it takes more than 4 seconds, it will be killed

		defer cancel()

		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			httpStatusCode = http.StatusBadRequest
			errMsg := "Must indicate flowName"
			http.Error(w, errMsg, httpStatusCode)

			duration := time.Since(startTime).Seconds()
			metrics.IncHTTPRequestsTotal(r.URL.Path, r.Method, httpStatusCode)
			metrics.ObserveHTTPRequestDuration(r.URL.Path, r.Method, httpStatusCode, duration)
			metrics.IncFlowExecution(flowName, "error_bad_request")

			return
		}

		flow, exists := flowRegistry[flowName]
		if !exists {
			httpStatusCode = http.StatusNotFound
			errMsg := fmt.Sprintf("Flow '%s' not found", flowName)
			http.Error(w, errMsg, httpStatusCode)

			// Errors Metrics
			duration := time.Since(startTime).Seconds()
			metrics.IncFlowExecuted(flowName, "error_flow_not_found")
			metrics.ObserveFlowDuration(flowName, "error_flow_not_found", duration)
			metrics.IncHTTPRequestsTotal(r.URL.Path, r.Method, httpStatusCode)
			metrics.ObserveHTTPRequestDuration(r.URL.Path, r.Method, httpStatusCode, duration)
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

		flowSucceeded := true
		for _, res := range results {
			if result, ok := res.(map[string]interface{}); ok {
				if _, hasError := result["error"]; hasError {
					flowSucceeded = false
					break
				}
			}
		}
		response["success"] = flowSucceeded

		if flowSucceeded {
			metrics.IncFlowExecuted(flowName, "success")
		} else {
			metrics.IncFlowExecuted(flowName, "error")
		}

		duration := time.Since(startTime).Seconds()
		metrics.ObserveFlowDuration(flowName, "success", duration)

		metrics.IncHTTPRequestsTotal(r.URL.Path, r.Method, httpStatusCode)
		metrics.ObserveHTTPRequestDuration(r.URL.Path, r.Method, httpStatusCode, duration)

		if httpStatusCode != http.StatusOK {
			w.WriteHeader(httpStatusCode)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.WithError(err).Error("Error encoding JSON response")
		}
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

		// Plugin duration metrics
		pluginStartTime := time.Now()
		var pluginStatusLabel string // State plugin: success or error

		plugin, err := pluginManager.GetPlugin(step.PluginRef)
		if err != nil {
			logger.Errorf("Plugin not found: %s - %v", step.PluginRef, err)
			results = append(results, map[string]interface{}{
				"plugin": step.PluginRef,
				"error":  fmt.Sprintf("Plugin not found: %v", err),
			})

			// --- Increase Executed Plugin Metric (CASE: Plugin NOT FOUND) ---
			// We consider "plugin not found" as a type of plugin execution error.
			pluginStatusLabel = "error_plugin_not_found"
			metrics.IncPluginExecuted(step.PluginRef, pluginStatusLabel)
			pluginDuration := time.Since(pluginStartTime).Seconds()
			metrics.ObservePluginDuration(step.PluginRef, pluginStatusLabel, pluginDuration) // --- End increase ---

			continue
		}

		for k, v := range step.Parameters {
			shared[k] = v
		}

		shared["previous_result"] = lastResult

		shared["_input"] = lastResult

		logger.Infof("Executing plugin: %s", step.PluginRef)
		res, errPluginExecute := plugin.Execute(ctx, r, &shared) // Call to the plugin

		if errPluginExecute != nil {
			pluginStatusLabel = "error_execution"
		} else {
			pluginStatusLabel = "success"
		}
		metrics.IncPluginExecuted(step.PluginRef, pluginStatusLabel)
		pluginDuration := time.Since(pluginStartTime).Seconds()
		metrics.ObservePluginDuration(step.PluginRef, pluginStatusLabel, pluginDuration)

		if errPluginExecute != nil {
			logger.Errorf("Error executing plugin: %s - %v", step.PluginRef, errPluginExecute)
			results = append(results, map[string]interface{}{
				"plugin": step.PluginRef,
				"error":  fmt.Sprintf("Error: %v", errPluginExecute),
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
		resultMap := map[string]interface{}{
			"plugin": step.PluginRef,
			"result": res,
		}

		// Only add formatted_result if it exists
		if formattedResult != "" {
			resultMap["formatted_result"] = formattedResult
		}

		results = append(results, resultMap)

		logMsg := fmt.Sprintf("Result from %s: ", step.PluginRef)

		if step.PluginRef == "formatter-plugin" {

			logMsg += "[long output, check the slack channel ;D]"
		} else {
			if len(formattedResult) > 100 {
				logMsg += formattedResult[:100] + "..."
			} else {
				logMsg += formattedResult
			}
		}
		logger.Info(logMsg)

		lastResult = res
	}

	return results
}
