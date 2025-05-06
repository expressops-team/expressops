package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// ActivePlugins measures currently active plugins
	ActivePlugins = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "expressops_active_plugins",
		Help: "Current number of active plugins",
	})

	// FlowsExecuted counts flow executions
	FlowsExecuted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "expressops_flows_executed_total",
		Help: "The total number of executed flows",
	}, []string{"flow_name", "status"})

	// FlowDuration measures flow execution time
	FlowDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "expressops_flow_duration_seconds",
		Help:    "Flow execution duration in seconds",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
	}, []string{"flow_name"})

	// PluginErrors counts plugin errors
	PluginErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "expressops_plugin_errors_total",
		Help: "The total number of plugin errors",
	}, []string{"plugin_name", "error_type"})

	// HttpRequests counts HTTP requests
	HttpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "expressops_http_requests_total",
		Help: "The total number of HTTP requests",
	}, []string{"endpoint", "method", "status"})

	// MemoryUsage measures memory usage
	MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "expressops_memory_usage_bytes",
		Help: "Current memory usage in bytes",
	})

	// CpuUsage measures CPU usage
	CpuUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "expressops_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})

	// PluginLatency measures plugin latency
	PluginLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "expressops_plugin_latency_seconds",
		Help:    "Plugin execution latency in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
	}, []string{"plugin_name"})

	// ConcurrentPlugins measures concurrently running plugins
	ConcurrentPlugins = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "expressops_concurrent_plugins",
		Help: "Number of plugins currently running",
	})

	// StorageOperations counts storage operations
	StorageOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "expressops_storage_operations_total",
		Help: "The total number of storage operations",
	}, []string{"operation", "status"})

	// StorageUsage measures storage usage
	StorageUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "expressops_storage_usage_bytes",
		Help: "Current storage usage in bytes",
	})
)

// MetricsHandler returns an HTTP handler for Prometheus metrics
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// SetActivePlugins updates the active plugins counter
func SetActivePlugins(count int) {
	ActivePlugins.Set(float64(count))
}

// RecordFlowExecution records a flow execution
func RecordFlowExecution(flowName string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	FlowsExecuted.WithLabelValues(flowName, status).Inc()
}

// RecordFlowDuration records a flow duration
func RecordFlowDuration(flowName string, duration time.Duration) {
	FlowDuration.WithLabelValues(flowName).Observe(duration.Seconds())
}

// RecordPluginError records a plugin error
func RecordPluginError(pluginName, errorType string) {
	PluginErrors.WithLabelValues(pluginName, errorType).Inc()
}

// RecordHttpRequest records an HTTP request
func RecordHttpRequest(endpoint, method string, statusCode int) {
	status := strconv.Itoa(statusCode)
	HttpRequests.WithLabelValues(endpoint, method, status).Inc()
}

// RecordMemoryUsage records memory usage
func RecordMemoryUsage(bytes float64) {
	MemoryUsage.Set(bytes)
}

// RecordCpuUsage records CPU usage
func RecordCpuUsage(percent float64) {
	CpuUsage.Set(percent)
}

// RecordPluginLatency records plugin latency
func RecordPluginLatency(pluginName string, duration time.Duration) {
	PluginLatency.WithLabelValues(pluginName).Observe(duration.Seconds())
}

// UpdateConcurrentPlugins updates the concurrent plugins counter
func UpdateConcurrentPlugins(delta int) {
	ConcurrentPlugins.Add(float64(delta))
}

// RecordStorageOperation records a storage operation
func RecordStorageOperation(operation string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	StorageOperations.WithLabelValues(operation, status).Inc()
}

// UpdateStorageUsage updates storage usage
func UpdateStorageUsage(bytes float64) {
	StorageUsage.Set(bytes)
}
