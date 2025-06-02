package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once     sync.Once
	registry *prometheus.Registry

	// Métricas de Prometheus
	ActivePlugins     prometheus.Gauge
	FlowsExecuted     *prometheus.CounterVec
	FlowDuration      *prometheus.HistogramVec
	PluginErrors      *prometheus.CounterVec
	HttpRequests      *prometheus.CounterVec
	MemoryUsage       prometheus.Gauge
	CpuUsage          prometheus.Gauge
	PluginLatency     *prometheus.HistogramVec
	ConcurrentPlugins prometheus.Gauge
	StorageOperations *prometheus.CounterVec
	StorageUsage      prometheus.Gauge
)

func init() {
	once.Do(func() {
		// Crear un registry personalizado
		registry = prometheus.NewRegistry()

		// Registrar los colectores por defecto (versión actualizada)
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		registry.MustRegister(collectors.NewGoCollector())

		// ActivePlugins measures currently active plugins
		ActivePlugins = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "expressops_active_plugins",
			Help: "Current number of active plugins",
		})
		registry.MustRegister(ActivePlugins)

		// FlowsExecuted counts flow executions
		FlowsExecuted = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "expressops_flows_executed_total",
			Help: "The total number of executed flows",
		}, []string{"flow_name", "status"})
		registry.MustRegister(FlowsExecuted)

		// FlowDuration measures flow execution time
		FlowDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "expressops_flow_duration_seconds",
			Help:    "Flow execution duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}, []string{"flow_name"})
		registry.MustRegister(FlowDuration)

		// PluginErrors counts plugin errors
		PluginErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "expressops_plugin_errors_total",
			Help: "The total number of plugin errors",
		}, []string{"plugin_name", "error_type"})
		registry.MustRegister(PluginErrors)

		// HttpRequests counts HTTP requests
		HttpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "expressops_http_requests_total",
			Help: "The total number of HTTP requests",
		}, []string{"endpoint", "method", "status"})
		registry.MustRegister(HttpRequests)

		// MemoryUsage measures memory usage
		MemoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "expressops_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		})
		registry.MustRegister(MemoryUsage)

		// CpuUsage measures CPU usage
		CpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "expressops_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		})
		registry.MustRegister(CpuUsage)

		// PluginLatency measures plugin latency
		PluginLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "expressops_plugin_latency_seconds",
			Help:    "Plugin execution latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		}, []string{"plugin_name"})
		registry.MustRegister(PluginLatency)

		// ConcurrentPlugins measures concurrently running plugins
		ConcurrentPlugins = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "expressops_concurrent_plugins",
			Help: "Number of plugins currently running",
		})
		registry.MustRegister(ConcurrentPlugins)

		// StorageOperations counts storage operations
		StorageOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "expressops_storage_operations_total",
			Help: "The total number of storage operations",
		}, []string{"operation", "status"})
		registry.MustRegister(StorageOperations)

		// StorageUsage measures storage usage
		StorageUsage = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "expressops_storage_usage_bytes",
			Help: "Current storage usage in bytes",
		})
		registry.MustRegister(StorageUsage)
	})
}

// MetricsHandler returns an HTTP handler for Prometheus metrics
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
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
