// internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"expressops/api/v1alpha1"
	"expressops/internal/metrics"
	pluginManager "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// registry of flows
var flowRegistry map[string]v1alpha1.Flow

// initializeFlowRegistry loads flows defined in the configuration file
func initializeFlowRegistry(cfg *v1alpha1.Config, logger *logrus.Logger) {
	flowRegistry = make(map[string]v1alpha1.Flow)
	for _, flow := range cfg.Flows {
		flowRegistry[flow.Name] = flow
		logger.Infof("Flow registered: %s", flow.Name)
	}
}

func StartServer(cfg *v1alpha1.Config, logger *logrus.Logger) {
	initializeFlowRegistry(cfg, logger)

	// Start resource monitoring routine (metrics will already be initialized by expressops.go)
	go monitorResourceUsage(logger)

	address := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)

	timeout := time.Duration(cfg.Server.TimeoutSec) * time.Second

	// ONLY one generic handler that will handle all flows
	http.HandleFunc("/flow", metricsMiddleware(dynamicFlowHandler(logger, timeout), logger))

	// Prometheus metrics endpoint
	http.Handle("/metrics", metrics.MetricsHandler())

	logger.Infof("Server listening on http://%s", address)
	logger.Infof("Prometheus metrics available at http://%s/metrics", address)

	// help for the user
	logger.Infof("➡️ curl http://%s/flow?flowName=<flow_name> ⬅️", address)

	srv := &http.Server{Addr: address}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Error starting server: %v", err)
	}
}

func monitorResourceUsage(logger *logrus.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Monitor actual resource metrics
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Update memory usage metrics
			metrics.RecordMemoryUsage(float64(m.Alloc))

			// Get CPU usage (simulated for development)
			cpuUsage := 25.0 + rand.Float64()*20.0 // Value between 25-45% for demo
			metrics.RecordCpuUsage(cpuUsage)

			// Update storage usage (simulated)
			storageUsed := 1024.0 * 1024.0 * (100.0 + rand.Float64()*50.0) // 100-150 MB for demo
			metrics.UpdateStorageUsage(storageUsed)

			// Update concurrent plugins count
			// This is just a placeholder - in real code this would be more dynamic
			activeConcurrentPlugins := 0
			// Some logic to count active plugins would go here
			metrics.UpdateConcurrentPlugins(activeConcurrentPlugins)

			logger.Debug("Updated resource usage metrics")
		}
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
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// Validate and get flow
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

		// Check for errors
		for _, res := range results {
			if result, ok := res.(map[string]interface{}); ok {
				if _, hasError := result["error"]; hasError {
					response["success"] = false
					break
				}
			}
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
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

// Find all steps that depend on the given step
func findDependentSteps(completedStep *stepExecution, execCtx *executionContext) []*stepExecution {
	globalPlanMutex.Lock()
	defer globalPlanMutex.Unlock()

	// Return the list of steps that directly depend on this one
	return globalDependents[completedStep]
}

// Build a dependency-aware execution plan from the pipeline
func buildExecutionPlan(pipeline []v1alpha1.Step, baseShared map[string]interface{}) []*stepExecution {
	execSteps := make([]*stepExecution, 0, len(pipeline))
	pluginRefToStep := make(map[string]*stepExecution)

	// First pass: Create step executions
	for i, step := range pipeline {
		// Skip commented plugins
		if step.PluginRef == "" {
			continue
		}

		// Create new shared context for this step
		stepShared := make(map[string]interface{})
		for k, v := range baseShared {
			stepShared[k] = v
		}

		// Add step parameters to shared context
		for k, v := range step.Parameters {
			stepShared[k] = v
		}

		execStep := &stepExecution{
			step:         step,
			index:        i,
			sharedCtx:    stepShared,
			dependencies: make([]*stepExecution, 0),
			executed:     false,
			hasError:     false,
		}

		execSteps = append(execSteps, execStep)
		pluginRefToStep[step.PluginRef] = execStep
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
