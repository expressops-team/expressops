package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counter for the total number of executed flows, labeled 'flowName'
	flowsExecutedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_flows_executed_total", // Metric name
			Help: "Total number of flows executed.", // Descriptive help
		},
		[]string{"flowName", "status"}, // Labels that the metric will have
	)

	// Histogram for flow execution duration
	flowExecutionDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "expressops_flow_execution_duration_seconds",
			Help:    "Duration of flow executions in seconds.",
			Buckets: prometheus.DefBuckets, // Puedes usar prometheus.ExponentialBuckets(0.001, 2, 15) para más granularidad
		},
		[]string{"flowName", "status"},
	)

	// Counter for total plugin executions, with 'pluginRef' and 'status' tags (success/error)
	pluginsExecutedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_plugins_executed_total",
			Help: "Total number of plugin executions attempted.",
		},
		[]string{"pluginRef", "status"},
	)

	// Histogram for plugin execution duration
	pluginExecutionDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "expressops_plugin_execution_duration_seconds",
			Help:    "Duration of plugin executions in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pluginRef", "status"},
	)

	// Counter for Slack notifications
	slackNotificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_slack_notifications_total",
			Help: "Total number of Slack notifications sent.",
		},
		[]string{"status", "channel"}, // 'status' could be "success" or "error"
	)

	// Counter for individuals health checks
	healthChecksPerformedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_health_checks_performed_total",
			Help: "Total number of individual health checks performed.",
		},
		[]string{"check_type", "status"}, // 'check_type' (cpu, mem, disk), 'status' (ok, fail)
	)

	// Gauge for resource usage (CPU, Memory, Disk most critical)
	// GaugeVec to be able to have different types of resources with labels
	resourceUsageGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "expressops_resource_usage_percent",
			Help: "Current resource usage percentage.",
		},
		[]string{"resource_type", "mount_point"}, // mount_point will be "" for cpu/mem
	)

	// Counter for Kubernetes probes (liveness/readiness)
	kubernetesProbesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_kubernetes_probes_total",
			Help: "Total number of Kubernetes liveness/readiness probes received.",
		},
		[]string{"probe_type", "path"}, // "liveness/readiness", "/healthz"
	)

	// Counter for user creation operations
	userCreationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_user_creation_total",
			Help: "Total number of user creation operations.",
		},
		[]string{"username", "status"}, // status: success, error, simulation
	)

	// Counter for permission changes
	permissionsChangesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_permissions_changes_total",
			Help: "Total number of permission changes.",
		},
		[]string{"path", "username", "status"}, // status: success, error, simulation
	)

	// Counter for message formatting operations
	formattingOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_formatting_operations_total",
			Help: "Total number of message formatting operations.",
		},
		[]string{"format_type", "status"}, // format_type: health_alert, etc.
	)

	// Histogram for sleep plugin duration
	sleepDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "expressops_sleep_duration_seconds",
			Help:    "Duration of sleep operations in seconds.",
			Buckets: []float64{1, 2, 5, 10, 30, 60}, // buckets for different sleep durations
		},
	)

	// Counter for test print operations
	testPrintTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_test_print_total",
			Help: "Total number of test print operations.",
		},
		[]string{"status"}, // status: success, error
	)
	// Histogram for HTTP request duration
	httpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "expressops_http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "code"},
	)

	// Counter for total HTTP requests
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"path", "method", "code"},
	)

	// Gauge for active flow handlers
	activeFlowHandlersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "expressops_active_flow_handlers_gauge",
			Help: "Number of currently active flow handlers.",
		},
	)
)

// --- PUBLIC FUNCTIONS TO ACCESS METRICS FROM OTHER PACKAGES ---

// IncFlowExecuted records the execution of a flow.
func IncFlowExecuted(flowName, status string) {
	flowsExecutedTotal.WithLabelValues(flowName, status).Inc()
}

func ObserveFlowDuration(flowName, status string, durationSeconds float64) {
	flowExecutionDurationSeconds.WithLabelValues(flowName, status).Observe(durationSeconds)
}

// IncPluginExecuted records the execution of a plugin.
func IncPluginExecuted(pluginRef, status string) {
	pluginsExecutedTotal.WithLabelValues(pluginRef, status).Inc()
}

// ObservePluginDuration records the duration of a plugin execution
func ObservePluginDuration(pluginRef, status string, durationSeconds float64) {
	pluginExecutionDurationSeconds.WithLabelValues(pluginRef, status).Observe(durationSeconds)
}

// IncSlackNotification records a Slack notification.
func IncSlackNotification(status, channel string) {
	slackNotificationsTotal.WithLabelValues(status, channel).Inc()
}

// IncHealthCheckPerformed registers an individual health check.
func IncHealthCheckPerformed(checkType, status string) {
	healthChecksPerformedTotal.WithLabelValues(checkType, status).Inc()
}

// SetResourceUsage records the percentage of usage of a resource.
func SetResourceUsage(resourceType, mountPoint string, usagePercent float64) {
	if mountPoint == "" && (resourceType == "cpu" || resourceType == "memory") {
		resourceUsageGauge.WithLabelValues(resourceType, "").Set(usagePercent)
	} else if resourceType == "disk" && mountPoint != "" {
		resourceUsageGauge.WithLabelValues(resourceType, mountPoint).Set(usagePercent)
	}
}

// IncKubernetesProbe is a convenience function for liveness probes.
func IncKubernetesProbe(probeType, path string) {
	kubernetesProbesTotal.WithLabelValues(probeType, path).Inc()
}

// IncUserCreation records a user creation operation.
func IncUserCreation(username, status string) {
	userCreationTotal.WithLabelValues(username, status).Inc()
}

// IncPermissionsChange records a permission change operation.
func IncPermissionsChange(path, username, status string) {
	permissionsChangesTotal.WithLabelValues(path, username, status).Inc()
}

// IncFormattingOperation records a formatting operation.
func IncFormattingOperation(formatType, status string) {
	formattingOperationsTotal.WithLabelValues(formatType, status).Inc()
}

// ObserveSleepDuration records the duration of a sleep operation.
func ObserveSleepDuration(seconds float64) {
	sleepDurationSeconds.Observe(seconds)
}

// IncTestPrint records a test print operation.
func IncTestPrint(status string) {
	testPrintTotal.WithLabelValues(status).Inc()
}

// ObserveHttpRequestDuration records the duration of an HTTP request.
func ObserveHttpRequestDuration(path, method string, code int, durationSeconds float64) {
	httpRequestDurationSeconds.WithLabelValues(path, method, fmt.Sprintf("%d", code)).Observe(durationSeconds)
}

// IncHttpRequestsTotal increments the counter for HTTP requests.
func IncHttpRequestsTotal(path, method string, code int) {
	httpRequestsTotal.WithLabelValues(path, method, fmt.Sprintf("%d", code)).Inc()
}

// IncActiveFlowHandlers increments the gauge for active flow handlers.
func IncActiveFlowHandlers() {
	activeFlowHandlersGauge.Inc()
}

// DecActiveFlowHandlers decrements the gauge for active flow handlers.
func DecActiveFlowHandlers() {
	activeFlowHandlersGauge.Dec()
}
