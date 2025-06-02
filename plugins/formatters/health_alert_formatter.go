package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"expressops/internal/metrics"
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

// Mensaje por defecto cuando no hay datos específicos de health
var defaultMessage = "No health data available. Please check system logs for more information."

type FormatterPlugin struct {
	logger     *logrus.Logger
	thresholds map[string]ThresholdLevels
	config     map[string]interface{}
}

// Initialize sets up the plugin
func (f *FormatterPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	f.logger = logger
	f.config = config
	pluginName := "HealthAlertFormatterPlugin" // Definir nombre para logs

	logFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "Initialize",
	}

	f.thresholds = make(map[string]ThresholdLevels)
	for k, v := range DefaultThresholds {
		f.thresholds[k] = v
	}

	if thresholdsConfig, ok := config["thresholds"].(map[string]interface{}); ok {
		logFields["thresholdsConfigured"] = true
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
				f.logger.WithFields(logFields).WithFields(logrus.Fields{
					"metricType":        metricType,
					"warningThreshold":  threshold.Warning,
					"criticalThreshold": threshold.Critical,
				}).Debug("Umbral personalizado configurado")
			}
		}
	} else {
		logFields["thresholdsConfigured"] = false
	}

	f.logger.WithFields(logFields).Info("HealthAlertFormatterPlugin inicializado")
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

// Execute formats health data for display and notification
func (f *FormatterPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	pluginName := "HealthAlertFormatterPlugin"
	flowName := request.URL.Query().Get("flowName")
	baseLogFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "Execute",
		"flowName":   flowName,
	}
	f.logger.WithFields(baseLogFields).Info("Formateando resultados de health check")

	if _, ok := (*shared)["_input"].(map[string]interface{}); !ok {
		f.logger.Error("No valid _input received")
		metrics.IncFormattingOperation("health_alert", "error_input")
		return "", fmt.Errorf("no valid _input received")
	}

	var (
		formattedMessage interface{}
		err              error
	)

	if kubeResults, ok := (*shared)["kube_health_results"].([]map[string]string); ok {
		f.logger.WithFields(baseLogFields).Info("Procesando datos de Kubernetes health")
		formattedMessage, err = f.formatKubernetesHealth(kubeResults, shared, baseLogFields)
	} else if prev, ok := (*shared)["previous_result"]; ok {
		f.logger.WithFields(baseLogFields).Info("Procesando 'previous_result'")
		if podResults, ok := prev.([]map[string]string); ok {
			f.logger.WithFields(baseLogFields).Info("Procesando resultados de pod desde plugin anterior")
			formattedMessage, err = f.formatKubernetesHealth(podResults, shared, baseLogFields)
		} else if healthMap, ok := prev.(map[string]interface{}); ok {
			f.logger.WithFields(baseLogFields).Info("Procesando datos generales de health desde plugin anterior")
			formattedMessage, err = f.formatHealthData(healthMap, shared, baseLogFields)
		} else {
			errMsg := "Formato de 'previous_result' no reconocido"
			f.logger.WithFields(baseLogFields).WithField("previousResultType", fmt.Sprintf("%T", prev)).Error(errMsg)
			formattedMessage = defaultMessage
			(*shared)["message"] = defaultMessage
			(*shared)["severity"] = "info" // Reset severity
		}
	} else {
		f.logger.WithFields(baseLogFields).Info("No se encontraron datos de health específicos, usando mensaje por defecto.")
		formattedMessage = defaultMessage
		(*shared)["message"] = defaultMessage
		(*shared)["severity"] = "info"
	}

	if err != nil {
		return nil, err
	}

	f.logger.WithFields(baseLogFields).WithField("formattedMessageLength", len(formattedMessage.(string))).Info("Formateo completado")
	return formattedMessage, nil
}

// formatHealthData formats general health check data
func (f *FormatterPlugin) formatHealthData(input map[string]interface{}, shared *map[string]any, baseLogFields logrus.Fields) (interface{}, error) {
	var logFormatted strings.Builder
	logFormatted.WriteString("Health check: ")
	var alertFormatted strings.Builder
	alertFormatted.WriteString("\n✨ Health Status Report ✨\n\n")

	if _, ok := input["health_status"].(map[string]string); !ok {
		f.logger.Error("Result without health_status field")
		metrics.IncFormattingOperation("health_alert", "error_status")
		return "", fmt.Errorf("health check result must contain a health_status field")
	}

	hasErrors := false

	sectionLogFields := make(logrus.Fields)
	for k, v := range baseLogFields {
		sectionLogFields[k] = v
	}
	sectionLogFields["dataType"] = "generalHealth"

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
				f.logger.WithFields(sectionLogFields).WithFields(logrus.Fields{"checkName": k, "checkStatus": v}).Warn("Health check fallido")
			}
		}
		logFormatted.WriteString(fmt.Sprintf("Checks:%s ", map[bool]string{true: "OK", false: "FAIL"}[checksOK]))
		alertFormatted.WriteString("\n")
	} else {
		f.logger.WithFields(sectionLogFields).Debug("No hay datos de 'health_status' disponibles")
		alertFormatted.WriteString("🔍 Health Checks: No check data available\n\n")
	}

	// CPU info
	if cpuInfo, ok := input["cpu"].(map[string]interface{}); ok {
		alertFormatted.WriteString("🖥️  CPU Usage:\n")
		if usage, ok := cpuInfo["usage_percent"].(float64); ok {
			cpuThreshold := f.thresholds["cpu"]
			if usage >= cpuThreshold.Critical || usage >= cpuThreshold.Warning {
				hasErrors = true
			}
			logFormatted.WriteString(fmt.Sprintf("CPU:%.1f%% ", usage))
			alertFormatted.WriteString(fmt.Sprintf("  Usage: %s\n", f.formatPercentage(usage, "cpu", false)))
		}
		alertFormatted.WriteString("\n")
	}

	// Memory info
	if memInfo, ok := input["memory"].(map[string]interface{}); ok {
		alertFormatted.WriteString("🧠 Memory Usage:\n")
		if usedPercent, ok := memInfo["used_percent"].(float64); ok {
			memThreshold := f.thresholds["memory"]
			if usedPercent >= memThreshold.Critical || usedPercent >= memThreshold.Warning {
				hasErrors = true
			}
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
		maxDiskUsage := 0.0
		var criticalMount string
		for mount, usageData := range diskInfo {
			if strings.HasPrefix(mount, "/snap") {
				continue
			}
			if u, ok := usageData.(map[string]interface{}); ok {
				alertFormatted.WriteString(fmt.Sprintf("  %s:\n", mount))
				if usedPercent, hasPercent := u["used_percent"].(float64); hasPercent {
					diskThreshold := f.thresholds["disk"]
					if usedPercent > maxDiskUsage {
						maxDiskUsage = usedPercent
						criticalMount = mount
					}
					if usedPercent >= diskThreshold.Critical || usedPercent >= diskThreshold.Warning {
						hasErrors = true
					}
					alertFormatted.WriteString(fmt.Sprintf("    Usage: %s\n", f.formatPercentage(usedPercent, "disk", false)))
				}
				if total, ok := u["total"].(uint64); ok {
					alertFormatted.WriteString(fmt.Sprintf("    Total: %s\n", f.formatSize(total)))
				}
				if free, ok := u["free"].(uint64); ok {
					alertFormatted.WriteString(fmt.Sprintf("    Free:  %s\n", f.formatSize(free)))
				}
			}
		}
		if maxDiskUsage > 0 {
			logFormatted.WriteString(fmt.Sprintf("Disk(%s):%.1f%% ", criticalMount, maxDiskUsage))
		}
	}

	alertFormatted.WriteString("\n")
	if hasErrors {
		logFormatted.WriteString("Status:WARNING")
		alertFormatted.WriteString("⚠️ Issues detected! Please check the output above.\n")

		metrics.IncFormattingOperation("health_alert", "warning")
	} else {
		logFormatted.WriteString("Status:OK")
		alertFormatted.WriteString("✅ All systems operational!\n")
		metrics.IncFormattingOperation("health_alert", "success")

	}

	message := alertFormatted.String()
	(*shared)["message"] = message

	f.logger.WithFields(sectionLogFields).WithFields(logrus.Fields{
		"logSummary":    logFormatted.String(),
		"finalSeverity": (*shared)["severity"],
	}).Debug("Resumen de log de health")

	return message, nil
}

// formatKubernetesHealth formats Kubernetes health check results
func (f *FormatterPlugin) formatKubernetesHealth(kubeResults []map[string]string, shared *map[string]any, baseLogFields logrus.Fields) (interface{}, error) {
	var sb strings.Builder
	totalPods := len(kubeResults)
	problemPods := 0

	sectionLogFields := make(logrus.Fields)
	for k, v := range baseLogFields {
		sectionLogFields[k] = v
	}
	sectionLogFields["dataType"] = "kubernetesHealth"

	f.logger.WithFields(sectionLogFields).WithField("podCount", totalPods).Info("Formateando datos de health de Kubernetes")

	sb.WriteString("\n🚢 Kubernetes Health Report 🚢\n\n")
	sb.WriteString("Pod Status:\n")

	for _, pod := range kubeResults {
		status := pod["status"]
		emoji := pod["emoji"]
		name := pod["name"]
		if emoji != "✅" {
			problemPods++
			f.logger.WithFields(sectionLogFields).WithFields(logrus.Fields{
				"podName":   name,
				"podStatus": status,
				"podEmoji":  emoji,
			}).Warn("Pod de Kubernetes con problemas")
		}
		sb.WriteString(fmt.Sprintf("  %s: %s %s\n", name, status, emoji))
	}

	sb.WriteString("\nSummary:\n")
	sb.WriteString(fmt.Sprintf("  Total pods: %d\n", totalPods))
	sb.WriteString(fmt.Sprintf("  Healthy pods: %d\n", totalPods-problemPods))

	if problemPods > 0 {
		sb.WriteString(fmt.Sprintf("  Problem pods: %d 🔴\n", problemPods))
		sb.WriteString("\n⚠️ Issues detected! Please check your Kubernetes cluster.\n")
		(*shared)["severity"] = "warning"
		f.logger.WithFields(sectionLogFields).WithFields(logrus.Fields{
			"problemPodCount": problemPods,
			"finalSeverity":   "warning",
		}).Warn("Problemas detectados en pods de Kubernetes")
	} else {
		sb.WriteString("\n✅ All pods are healthy!\n")
		(*shared)["severity"] = "info"
		f.logger.WithFields(sectionLogFields).WithFields(logrus.Fields{
			"finalSeverity": "info",
		}).Info("Todos los pods de Kubernetes saludables")
	}

	message := sb.String()
	(*shared)["message"] = message
	return message, nil
}

// FormatResult returns a simple string representation
func (f *FormatterPlugin) FormatResult(result interface{}) (string, error) {
	baseLogFields := logrus.Fields{
		"pluginName": "HealthAlertFormatterPlugin",
		"action":     "FormatResult",
	}
	f.logger.WithFields(baseLogFields).Debug("Formateando resultado")
	if result == nil {
		return "No result to format", nil
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// Export the plugin instance
var PluginInstance pluginconf.Plugin = &FormatterPlugin{}
