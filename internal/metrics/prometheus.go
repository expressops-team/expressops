package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Contador de flujos ejecutados
	FlowsExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_flows_executed_total",
			Help: "The total number of flows executed",
		},
		[]string{"flow_name", "status"},
	)

	// Histograma de duración de ejecución de flujos
	FlowDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "expressops_flow_duration_seconds",
			Help:    "Flow execution duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"flow_name"},
	)

	// Gauge para plugins activos
	ActivePlugins = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "expressops_active_plugins",
			Help: "The number of active plugins",
		},
	)

	// Contador de errores por plugin
	PluginErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_plugin_errors_total",
			Help: "The total number of plugin errors",
		},
		[]string{"plugin_name", "error_type"},
	)
)

// RecordFlowExecution registra la ejecución de un flujo
func RecordFlowExecution(flowName string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	FlowsExecuted.WithLabelValues(flowName, status).Inc()
	FlowDuration.WithLabelValues(flowName).Observe(duration.Seconds())
}

// RecordPluginError registra un error en un plugin
func RecordPluginError(pluginName, errorType string) {
	PluginErrors.WithLabelValues(pluginName, errorType).Inc()
}

// SetActivePlugins establece el número de plugins activos
func SetActivePlugins(count int) {
	ActivePlugins.Set(float64(count))
}

// MetricsHandler devuelve un handler HTTP para exponer métricas
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
