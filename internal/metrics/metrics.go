package metrics

import (
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
		[]string{"flowName"}, // Labels that the metric will have
	)

	// Counter for total plugin executions, with 'pluginRef' and 'status' tags (success/error)
	pluginsExecutedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expressops_plugins_executed_total",
			Help: "Total number of plugin executions attempted.",
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
)

// --- PUBLIC FUNCTIONS TO ACCESS METRICS FROM OTHER PACKAGES ---

// IncFlowExecuted records the execution of a flow.
func IncFlowExecuted(flowName string) {
	flowsExecutedTotal.WithLabelValues(flowName).Inc()
}

// IncPluginExecuted records the execution of a plugin.
func IncPluginExecuted(pluginRef, status string) {
	pluginsExecutedTotal.WithLabelValues(pluginRef, status).Inc()
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
