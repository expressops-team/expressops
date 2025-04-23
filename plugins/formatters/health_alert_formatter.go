package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// ThresholdLevels defines the threshold levels for metrics
type ThresholdLevels struct {
	Warning  float64
	Critical float64
}

// DefaultThresholds provides default values for different metrics
var DefaultThresholds = map[string]ThresholdLevels{
	"cpu":    {Warning: 50, Critical: 80},
	"memory": {Warning: 50, Critical: 80},
	"disk":   {Warning: 80, Critical: 90},
}

type FormatterPlugin struct {
	logger     *logrus.Logger
	thresholds map[string]ThresholdLevels
	config     map[string]interface{}
}

func (f *FormatterPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	f.logger = logger
	f.config = config
	f.logger.Info("Initializing Health Formatter Plugin")

	// Initialize thresholds with defaults
	f.thresholds = make(map[string]ThresholdLevels)
	for k, v := range DefaultThresholds {
		f.thresholds[k] = v
	}

	// Override with config if provided
	if thresholdsConfig, ok := config["thresholds"].(map[string]interface{}); ok {
		for metricType, thresholdValues := range thresholdsConfig {
			if values, ok := thresholdValues.(map[string]interface{}); ok {
				threshold := ThresholdLevels{}

				if warning, ok := values["warning"].(float64); ok {
					threshold.Warning = warning
				}

				if critical, ok := values["critical"].(float64); ok {
					threshold.Critical = critical
				}

				f.thresholds[metricType] = threshold
			}
		}
	}

	return nil
}

func (f *FormatterPlugin) formatPercentage(value float64, metricType string, forLog bool) string {
	threshold, exists := f.thresholds[metricType]
	if !exists {
		return fmt.Sprintf("%.2f%%", value)
	}

	if forLog {
		return fmt.Sprintf("%.2f%%", value)
	}

	if value >= threshold.Critical {
		return fmt.Sprintf("%.2f%% 🔴 CRITICAL", value)
	} else if value >= threshold.Warning {
		return fmt.Sprintf("%.2f%% 🟠 WARNING", value)
	}
	return fmt.Sprintf("%.2f%%", value)
}

func (f *FormatterPlugin) formatSize(value uint64) string {
	return fmt.Sprintf("%.2f GB", float64(value)/1024/1024/1024)
}

func (f *FormatterPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	f.logger.Info("Formatting health check results")

	// Always set a default message
	defaultMessage := "Health check formatting completed"
	(*shared)["message"] = defaultMessage

	// First check for previous plugin results directly
	if prev, ok := (*shared)["previous_result"]; ok && prev != nil {
		f.logger.Infof("Received previous result of type: %T", prev)

		// Handle map[string]string format that kubehealth might provide
		if podResults, ok := prev.([]map[string]string); ok {
			f.logger.Info("Detected pod results format, processing as Kubernetes data")
			return f.formatKubernetesHealth(podResults, shared)
		}
	}

	// Check for Kubernetes health data
	if kubeResults, ok := (*shared)["kube_health_results"].([]map[string]string); ok {
		f.logger.Info("Processing Kubernetes health data from shared context")
		return f.formatKubernetesHealth(kubeResults, shared)
	}

	// Try parsing input from shared map
	var input map[string]interface{}

	// Try several sources for input data
	if inputData, ok := (*shared)["_input"].(map[string]interface{}); ok {
		input = inputData
	} else if inputData, ok := (*shared)["input"].(map[string]interface{}); ok {
		input = inputData
	} else if inputData, ok := (*shared)["previous_result"].(map[string]interface{}); ok {
		input = inputData
	} else {
		// Create a basic message if no data found
		message := "No valid input data found. Creating default message."
		f.logger.Info(message)
		(*shared)["message"] = message
		return message, nil
	}

	// Process the input data
	return f.formatHealthData(input, shared)
}

// formatHealthData formats general health check data
func (f *FormatterPlugin) formatHealthData(input map[string]interface{}, shared *map[string]any) (interface{}, error) {
	// Simple log format (single line)
	var logFormatted strings.Builder
	logFormatted.WriteString("Health check: ")

	// Rich format for alerts/display
	var alertFormatted strings.Builder
	// Clean output ;)
	alertFormatted.WriteString("\n✨ Health Status Report ✨\n\n")

	hasErrors := false

	// Process health status checks if available
	if status, ok := input["health_status"].(map[string]string); ok {
		checksOK := true
		alertFormatted.WriteString("🔍 Health Checks:\n")
		for k, v := range status {
			if v == "OK" {
				alertFormatted.WriteString(fmt.Sprintf("  %s: ✅ OK\n", k))
			} else {
				checksOK = false
				hasErrors = true
				alertFormatted.WriteString(fmt.Sprintf("  %s: ❌ %s\n", k, v))
			}
		}

		// Add checks status to log
		if checksOK {
			logFormatted.WriteString("Checks:OK ")
		} else {
			logFormatted.WriteString("Checks:FAIL ")
		}

		alertFormatted.WriteString("\n")
	} else {
		// No health status, just show a message
		alertFormatted.WriteString("🔍 Health Checks: No check data available\n\n")
	}

	// CPU info
	if cpuInfo, ok := input["cpu"].(map[string]interface{}); ok {
		alertFormatted.WriteString("🖥️  CPU Usage:\n")
		if usage, ok := cpuInfo["usage_percent"].(float64); ok {
			// Check thresholds for errors only
			cpuThreshold := f.thresholds["cpu"]
			if usage >= cpuThreshold.Critical || usage >= cpuThreshold.Warning {
				hasErrors = true
			}

			// Simpler log format without status indicators
			logFormatted.WriteString(fmt.Sprintf("CPU:%.1f%% ", usage))
			alertFormatted.WriteString(fmt.Sprintf("  Usage: %s\n", f.formatPercentage(usage, "cpu", false)))
		}
		alertFormatted.WriteString("\n")
	}

	// Memory info
	if memInfo, ok := input["memory"].(map[string]interface{}); ok {
		alertFormatted.WriteString("🧠 Memory Usage:\n")

		if usedPercent, ok := memInfo["used_percent"].(float64); ok {
			// Check thresholds for errors only
			memThreshold := f.thresholds["memory"]
			if usedPercent >= memThreshold.Critical || usedPercent >= memThreshold.Warning {
				hasErrors = true
			}

			// Simpler log format without status indicators
			logFormatted.WriteString(fmt.Sprintf("Mem:%.1f%% ", usedPercent))
			alertFormatted.WriteString(fmt.Sprintf("  Usage: %s\n", f.formatPercentage(usedPercent, "memory", false)))
		}

		if total, ok := memInfo["total"].(uint64); ok {
			alertFormatted.WriteString(fmt.Sprintf("  Total: %s\n", f.formatSize(total)))
		}
		if used, ok := memInfo["used"].(uint64); ok {
			alertFormatted.WriteString(fmt.Sprintf("  Used:  %s\n", f.formatSize(used)))
		}
		if free, ok := memInfo["free"].(uint64); ok {
			alertFormatted.WriteString(fmt.Sprintf("  Free:  %s\n", f.formatSize(free)))
		}

		alertFormatted.WriteString("\n")
	}

	// Disk info
	if diskInfo, ok := input["disk"].(map[string]interface{}); ok {
		alertFormatted.WriteString("💽 Disk Usage:\n")

		// Track most critical disk usage
		maxDiskUsage := 0.0
		var criticalMount string

		for mount, usage := range diskInfo {
			// Skip snap mounts to reduce spam
			if strings.HasPrefix(mount, "/snap") {
				continue
			}

			if u, ok := usage.(map[string]interface{}); ok {
				alertFormatted.WriteString(fmt.Sprintf("  %s:\n", mount))

				usedPercent, hasPercent := u["used_percent"].(float64)
				if hasPercent {
					// Check disk threshold
					diskThreshold := f.thresholds["disk"]

					// Find the most critical disk
					if usedPercent > maxDiskUsage {
						maxDiskUsage = usedPercent
						criticalMount = mount

						// Set error flag based on thresholds
						if usedPercent >= diskThreshold.Critical || usedPercent >= diskThreshold.Warning {
							hasErrors = true
						}
					}

					// Format output for this disk
					diskPercent := usedPercent
					var diskLine string
					if diskPercent >= diskThreshold.Critical {
						diskLine = fmt.Sprintf("    Usage: %.2f%% 🔴 CRITICAL\n", diskPercent)
					} else if diskPercent >= diskThreshold.Warning {
						diskLine = fmt.Sprintf("    Usage: %.2f%% 🟠 WARNING\n", diskPercent)
					} else {
						diskLine = fmt.Sprintf("    Usage: %.2f%%\n", diskPercent)
					}
					alertFormatted.WriteString(diskLine)
				}

				if total, ok := u["total"].(uint64); ok {
					alertFormatted.WriteString(fmt.Sprintf("    Total: %s\n", f.formatSize(total)))
				}
				if free, ok := u["free"].(uint64); ok {
					alertFormatted.WriteString(fmt.Sprintf("    Free:  %s\n", f.formatSize(free)))
				}
			}
		}

		// Add most critical disk to log line (without status)
		if maxDiskUsage > 0 {
			logFormatted.WriteString(fmt.Sprintf("Disk:%s:%.1f%% ",
				criticalMount, maxDiskUsage))
		}
	}

	// Summary
	alertFormatted.WriteString("\n")
	if hasErrors {
		logFormatted.WriteString("Status:WARNING")
		alertFormatted.WriteString("⚠️ Issues detected! Please check the output above.\n")
	} else {
		logFormatted.WriteString("Status:OK")
		alertFormatted.WriteString("✅ All systems operational!\n")
	}

	// Set message in shared context for slack notification
	message := alertFormatted.String()
	(*shared)["message"] = message

	// Set severity based on errors
	if hasErrors {
		(*shared)["severity"] = "warning"
	} else {
		(*shared)["severity"] = "info"
	}

	return message, nil
}

// formatKubernetesHealth formats Kubernetes health check results
func (f *FormatterPlugin) formatKubernetesHealth(kubeResults []map[string]string, shared *map[string]any) (interface{}, error) {
	var sb strings.Builder
	hasIssues := false
	totalPods := len(kubeResults)
	problemPods := 0

	sb.WriteString("\n🚢 Kubernetes Health Report 🚢\n\n")
	sb.WriteString("Pod Status:\n")

	// Count issues and format each pod
	for _, pod := range kubeResults {
		status := pod["status"]
		emoji := pod["emoji"]

		if emoji != "✅" {
			hasIssues = true
			problemPods++
		}

		sb.WriteString(fmt.Sprintf("  %s: %s %s\n", pod["name"], status, emoji))
	}

	// Add summary
	sb.WriteString("\nSummary:\n")
	sb.WriteString(fmt.Sprintf("  Total pods: %d\n", totalPods))
	sb.WriteString(fmt.Sprintf("  Healthy pods: %d\n", totalPods-problemPods))

	if problemPods > 0 {
		sb.WriteString(fmt.Sprintf("  Problem pods: %d 🔴\n", problemPods))
		sb.WriteString("\n⚠️ Issues detected! Please check your Kubernetes cluster.\n")
	} else {
		sb.WriteString("\n✅ All pods are healthy!\n")
	}

	// Store in shared context for slack notification
	message := sb.String()
	(*shared)["message"] = message

	// Make sure severity is set
	if hasIssues {
		(*shared)["severity"] = "warning"
	} else {
		(*shared)["severity"] = "info"
	}

	return message, nil
}

func (f *FormatterPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No result to format", nil
	}

	if str, ok := result.(string); ok {
		return str, nil
	}

	return fmt.Sprintf("%v", result), nil
}

func NewFormatterPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &FormatterPlugin{
		logger:     logger,
		thresholds: make(map[string]ThresholdLevels),
	}
}

var PluginInstance = NewFormatterPlugin(logrus.New())
