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
		return
	}

	execCtx.logger.Infof("Executing plugin: %s", step.step.PluginRef)
	res, err := plugin.Execute(execCtx.ctx, execCtx.request, &step.sharedCtx)
	if err != nil {
		markStepFailed(step, execCtx, fmt.Sprintf("Error: %v", err))
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
