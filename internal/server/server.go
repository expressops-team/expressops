// Package server provides HTTP server functionality for the application
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

	// Start resource monitoring routine (metrics will already be initialized by expressops.go)
	go monitorResourceUsage(logger)

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
	http.HandleFunc("/flow", metricsMiddleware(dynamicFlowHandler(logger, timeout), logger))

	// Prometheus metrics endpoint
	http.Handle("/metrics", metrics.MetricsHandler())

	logger.Infof("Server listening on http://%s", address)
	logger.Infof("Prometheus metrics available at http://%s/metrics", address)

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

func monitorResourceUsage(logger *logrus.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Execute health-check-plugin to get real metrics
		plugin, err := pluginManager.GetPlugin("health-check-plugin")
		if err != nil {
			logger.Warnf("Error getting health-check-plugin: %v", err)
			continue
		}

		// Create a context and dummy request for the plugin
		ctx := context.Background()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		shared := make(map[string]interface{})

		// Execute the plugin to get real metrics
		result, err := plugin.Execute(ctx, req, &shared)
		if err != nil {
			logger.Warnf("Error executing health-check-plugin: %v", err)
			continue
		}

		// Convert result to a map
		healthData, ok := result.(map[string]interface{})
		if !ok {
			logger.Warn("Unexpected health check result format")
			continue
		}

		// Extract and update CPU metrics
		if cpuInfo, ok := healthData["cpu"].(map[string]interface{}); ok {
			if cpuPercent, ok := cpuInfo["usage_percent"].(float64); ok {
				metrics.RecordCpuUsage(cpuPercent)
				logger.Debugf("Updated CPU usage: %.2f%%", cpuPercent)
			}
		}

		// Extract and update memory metrics
		if memInfo, ok := healthData["memory"].(map[string]interface{}); ok {
			if used, ok := memInfo["used"].(uint64); ok {
				metrics.RecordMemoryUsage(float64(used))
				logger.Debugf("Updated memory usage: %d bytes", used)
			}
		}

		// Extract and update disk metrics
		if diskInfo, ok := healthData["disk"].(map[string]interface{}); ok {
			// Sum used space across all partitions
			var totalUsed uint64 = 0
			for _, partInfo := range diskInfo {
				if partData, ok := partInfo.(map[string]interface{}); ok {
					if used, ok := partData["used"].(uint64); ok {
						totalUsed += used
					}
				}
			}
			metrics.UpdateStorageUsage(float64(totalUsed))
			logger.Debugf("Updated storage usage: %d bytes", totalUsed)
		}

		// Update active plugins count
		activePlugins := 0
		globalPlanMutex.Lock()
		// Count plugins currently executing
		for _, steps := range globalStepPlan {
			for _, step := range steps {
				if !step.executed {
					activePlugins++
				}
			}
		}
		globalPlanMutex.Unlock()
		metrics.UpdateConcurrentPlugins(activePlugins)
		metrics.SetActivePlugins(activePlugins)

		logger.Debug("Updated all resource metrics from health-check-plugin")
	}
}

// Middleware to record Prometheus metrics
func metricsMiddleware(next http.HandlerFunc, logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flowName := r.URL.Query().Get("flowName")
		if flowName == "" {
			flowName = "unknown"
		}

		startTime := time.Now()

		// Create wrapper to capture status code
		mw := newMetricsResponseWriter(w)

		// Record HTTP request started
		metrics.RecordHttpRequest(r.URL.Path, r.Method, 200)

		// Call original handler
		next(mw, r)

		// Record metrics
		duration := time.Since(startTime)
		success := mw.statusCode < 400

		// Record flow execution with success status
		metrics.RecordFlowExecution(flowName, success)

		// Record flow duration
		metrics.RecordFlowDuration(flowName, duration)

		// Record final HTTP status
		metrics.RecordHttpRequest(r.URL.Path, r.Method, mw.statusCode)

		logger.WithFields(logrus.Fields{
			"flow":        flowName,
			"duration_ms": duration.Milliseconds(),
			"status_code": mw.statusCode,
		}).Debug("Flow execution metrics recorded")
	}
}

// Wrapper for ResponseWriter that captures status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{w, http.StatusOK}
}

func (mw *metricsResponseWriter) WriteHeader(code int) {
	mw.statusCode = code
	mw.ResponseWriter.WriteHeader(code)
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

		// Validate and get flow
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

		// Log execution info
		logger.WithFields(logrus.Fields{
			"flow": flowName, "ip": r.RemoteAddr,
		}).Info("Executing flow")

		// Process params
		params := parseParams(r.URL.Query().Get("params"))
		isAllFlowsFlow := flowName == "all-flows"

		// Execute and prepare response
		results := executeFlow(ctx, flow, params, r, logger, isAllFlowsFlow)
		response := map[string]interface{}{
			"flow": flowName, "success": true, "count": len(results),
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

	for _, pair := range strings.Split(paramsRaw, ";") {
		if kv := strings.SplitN(pair, ":", 2); len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

// Represents a step in the pipeline with its execution context
// https://github.com/saantiaguilera/go-pipeline/blob/master/step.go
type stepExecution struct {
	step         v1alpha1.Step
	index        int
	sharedCtx    map[string]interface{} //dependencies, result, flags for execution
	result       interface{}
	dependencies []*stepExecution
	executed     bool
	hasError     bool
}

// Global registry to track dependencies between steps for the current execution
var (
	globalPlanMutex sync.Mutex
	globalStepPlan  map[string][]*stepExecution
	// Track dependencies - which steps depend on this one
	globalDependents map[*stepExecution][]*stepExecution
)

// Initialize the global execution plan trackers
func initializeExecutionPlanTrackers(plan []*stepExecution) {
	globalPlanMutex.Lock()
	defer globalPlanMutex.Unlock()

	globalStepPlan = make(map[string][]*stepExecution)
	globalDependents = make(map[*stepExecution][]*stepExecution)

	for _, step := range plan {
		// Register step
		pluginRef := step.step.PluginRef
		globalStepPlan[pluginRef] = append(globalStepPlan[pluginRef], step)

		// Create dependents list if needed
		if _, exists := globalDependents[step]; !exists {
			globalDependents[step] = make([]*stepExecution, 0)
		}

		// Register reverse dependencies
		for _, dep := range step.dependencies {
			if _, exists := globalDependents[dep]; !exists {
				globalDependents[dep] = make([]*stepExecution, 0)
			}
			globalDependents[dep] = append(globalDependents[dep], step)
		}
	}
}

// buildExecutionPlan creates a plan of steps to execute from the flow pipeline
func buildExecutionPlan(pipeline []v1alpha1.Step, shared map[string]interface{}) []*stepExecution {
	var execSteps []*stepExecution
	pluginRefToStep := make(map[string]*stepExecution)

	// First pass: Create step objects
	for i, step := range pipeline {
		if step.PluginRef == "" {
			continue // Skip commented out steps
		}

		stepCtx := make(map[string]interface{})
		// Copy the shared context
		for k, v := range shared {
			stepCtx[k] = v
		}

		// Copy step parameters
		for k, v := range step.Parameters {
			stepCtx[k] = v
		}

		exec := &stepExecution{
			step:         step,
			index:        i,
			sharedCtx:    stepCtx,
			dependencies: make([]*stepExecution, 0),
		}

		execSteps = append(execSteps, exec)
		pluginRefToStep[step.PluginRef] = exec
	}

	// Second pass: Resolve dependencies
	for i, execStep := range execSteps {
		// Check for explicit dependencies in YAML config
		if len(execStep.step.DependsOn) > 0 {
			// Process explicit dependencies
			for _, depRef := range execStep.step.DependsOn {
				if depStep, exists := pluginRefToStep[depRef]; exists {
					execStep.dependencies = append(execStep.dependencies, depStep)
				}
			}
		} else if i > 0 && !execStep.step.Parallel {
			// Fallback: This step depends on the previous one if not marked as parallel
			// and has no explicit dependencies
			execStep.dependencies = append(execStep.dependencies, execSteps[i-1])
		}
	}

	// Initialize the global trackers
	initializeExecutionPlanTrackers(execSteps)

	return execSteps
}

// Execution context shared across all steps
type executionContext struct {
	ctx      context.Context
	logger   *logrus.Logger
	request  *http.Request
	wg       *sync.WaitGroup
	mutex    *sync.Mutex
	results  *[]interface{}
	allFlows bool
}

// Execute all steps in the plan, respecting dependencies
func executeSteps(plan []*stepExecution, execCtx *executionContext) {
	// Start all steps (no pending dependencies)
	for _, step := range plan {
		if len(step.dependencies) == 0 {
			execCtx.wg.Add(1)
			go executeStepAsync(step, execCtx)
		}
	}
}

// Execute a single step asynchronously
func executeStepAsync(step *stepExecution, execCtx *executionContext) {
	defer execCtx.wg.Done()

	// Increment concurrent plugins counter
	metrics.UpdateConcurrentPlugins(1)
	defer metrics.UpdateConcurrentPlugins(-1) // Decrease counter when done

	// Wait for dependencies in parallel
	var depWg sync.WaitGroup
	depErr := false

	for _, dep := range step.dependencies {
		depWg.Add(1)
		go func(dependency *stepExecution) {
			defer depWg.Done()
			for !dependency.executed {
				time.Sleep(5 * time.Millisecond)
			}
			if dependency.hasError {
				depErr = true
			}
		}(dep)
	}

	depWg.Wait()

	// Skip if any dependency failed
	if depErr {
		markStepFailed(step, execCtx, "Skipped due to dependency failure")
		metrics.RecordPluginError(step.step.PluginRef, "dependency_failure")
		return
	}

	// Add dependency results to context
	for _, dep := range step.dependencies {
		step.sharedCtx[fmt.Sprintf("%s_result", dep.step.PluginRef)] = dep.result
		step.sharedCtx["previous_result"] = dep.result // backward compatibility
		step.sharedCtx["_input"] = dep.result
	}

	// Get and execute plugin
	plugin, err := pluginManager.GetPlugin(step.step.PluginRef)
	if err != nil {
		markStepFailed(step, execCtx, fmt.Sprintf("Plugin not found: %v", err))
		metrics.RecordPluginError(step.step.PluginRef, "plugin_not_found")
		return
	}

	// Start measuring plugin execution time
	pluginStartTime := time.Now()

	execCtx.logger.Infof("Executing plugin: %s", step.step.PluginRef)
	res, err := plugin.Execute(execCtx.ctx, execCtx.request, &step.sharedCtx)

	// Record plugin execution latency
	pluginDuration := time.Since(pluginStartTime)
	metrics.RecordPluginLatency(step.step.PluginRef, pluginDuration)

	if err != nil {
		markStepFailed(step, execCtx, fmt.Sprintf("Error: %v", err))
		metrics.RecordPluginError(step.step.PluginRef, "execution_error")
		return
	}

	// Format result
	var formattedResult string
	if res != nil {
		formattedResult, err = plugin.FormatResult(res)
		if err != nil {
			execCtx.logger.Warnf("Format error: %v", err)
			formattedResult = fmt.Sprintf("%v", res)
		}
	}

	// Log and store result
	logResult(step.step.PluginRef, formattedResult, execCtx)

	execCtx.mutex.Lock()
	result := map[string]interface{}{
		"plugin": step.step.PluginRef,
		"result": res,
	}
	if formattedResult != "" {
		result["formatted_result"] = formattedResult
	}
	*execCtx.results = append(*execCtx.results, result)
	execCtx.mutex.Unlock()

	// Mark complete and trigger dependents
	step.result = res
	step.executed = true
	triggerDependentSteps(step, execCtx)
}

// Helper to mark a step as failed
func markStepFailed(step *stepExecution, execCtx *executionContext, errMsg string) {
	execCtx.logger.Errorf("Plugin %s: %s", step.step.PluginRef, errMsg)

	// Registrar error en métricas
	errorType := "execution_error"
	if strings.Contains(errMsg, "Plugin not found") {
		errorType = "plugin_not_found"
	} else if strings.Contains(errMsg, "Skipped due to dependency") {
		errorType = "dependency_failure"
	}
	metrics.RecordPluginError(step.step.PluginRef, errorType)

	execCtx.mutex.Lock()
	*execCtx.results = append(*execCtx.results, map[string]interface{}{
		"plugin": step.step.PluginRef,
		"error":  errMsg,
	})
	execCtx.mutex.Unlock()

	step.executed = true
	step.hasError = true
	triggerDependentSteps(step, execCtx)
}

// Start execution of steps that were waiting on this step
func triggerDependentSteps(completedStep *stepExecution, execCtx *executionContext) {
	// Find all steps that were waiting on this one
	for _, step := range findDependentSteps(completedStep, execCtx) {
		// Check if all dependencies are now satisfied
		allDepsComplete := true
		for _, dep := range step.dependencies {
			if !dep.executed {
				allDepsComplete = false
				break
			}
		}
		// If all dependencies are complete, start this step
		if allDepsComplete {
			execCtx.wg.Add(1)
			go executeStepAsync(step, execCtx)
		}
	}
}

// findDependentSteps returns all steps that depend on the given step
func findDependentSteps(step *stepExecution, execCtx *executionContext) []*stepExecution {
	globalPlanMutex.Lock()
	defer globalPlanMutex.Unlock()

	// Return the steps that depend on this one
	if deps, exists := globalDependents[step]; exists {
		return deps
	}

	// If no dependencies found, return empty slice
	return []*stepExecution{}
}

// step by step execution of the flow with dependency management
func executeFlow(ctx context.Context, flow v1alpha1.Flow, params map[string]interface{}, r *http.Request, logger *logrus.Logger, isAllFlowsFlow bool) []interface{} {
	var results []interface{}

	// Skip empty pipelines
	if len(flow.Pipeline) == 0 {
		return results
	}

	// Setup shared context
	shared := make(map[string]interface{})
	for k, v := range params {
		shared[k] = v
	}
	shared["flow_registry"] = flowRegistry

	// Prepare execution
	executionPlan := buildExecutionPlan(flow.Pipeline, shared)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	execCtx := &executionContext{
		ctx:      ctx,
		logger:   logger,
		request:  r,
		wg:       &wg,
		mutex:    &mutex,
		results:  &results,
		allFlows: isAllFlowsFlow,
	}

	// Run and wait for completion
	executeSteps(executionPlan, execCtx)
	wg.Wait()

	return results
}

// Helper function to log plugin results with appropriate formatting
func logResult(pluginRef string, formattedResult string, execCtx *executionContext) {
	switch {
	case strings.HasSuffix(pluginRef, "-formatter") || pluginRef == "formatter-plugin":
		execCtx.logger.Infof("Result from %s: [long output]", pluginRef)

	case strings.HasPrefix(formattedResult, "__MULTILINE_LOG__"):
		execCtx.logger.Infof("Result from %s (multi-line):", pluginRef)
		for _, line := range strings.Split(formattedResult, "__MULTILINE_LOG__") {
			if line != "" {
				execCtx.logger.Info(line)
			}
		}

	default:
		if !execCtx.allFlows && len(formattedResult) > 100 {
			execCtx.logger.Infof("Result from %s: %s...", pluginRef, formattedResult[:100])
		} else {
			execCtx.logger.Infof("Result from %s: %s", pluginRef, formattedResult)
		}
	}
}
