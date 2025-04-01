package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type FormatterPlugin struct {
	logger *logrus.Logger
}

func (f *FormatterPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	f.logger = logger
	f.logger.Info("Initializing Health Formatter Plugin")
	return nil
}

func (f *FormatterPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	f.logger.Info("Formatting health check results")

	input, ok := (*shared)["_input"].(map[string]interface{})
	if !ok {
		f.logger.Error("No valid _input received")
		return "", fmt.Errorf("no valid _input received")
	}

	var formatted strings.Builder
	// Clean output ;)
	formatted.WriteString("\n✨ Health Status Report ✨\n\n")

	status, ok := input["health_status"].(map[string]string)
	if !ok {
		f.logger.Error("Result without health_status field")
		return "", fmt.Errorf("health check result must contain a health_status field")
	}

	hasErrors := false
	formatted.WriteString("🔍 Health Checks:\n")
	for k, v := range status {
		if v == "OK" {
			formatted.WriteString(fmt.Sprintf("  %s: ✅ OK\n", k))
		} else {
			hasErrors = true
			formatted.WriteString(fmt.Sprintf("  %s: ❌ %s\n", k, v))
		}
	}
	formatted.WriteString("\n")

	// CPU info
	if cpuInfo, ok := input["cpu"].(map[string]interface{}); ok {
		formatted.WriteString("🖥️  CPU Usage:\n")
		if usage, ok := cpuInfo["usage_percent"].(float64); ok {
			if usage > 80 {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%% (HIGH)\n", usage))
			} else if usage > 50 {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%% (MEDIUM)\n", usage))
			} else {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%%\n", usage))
			}
		}
		formatted.WriteString("\n")
	}

	// Memory info
	if memInfo, ok := input["memory"].(map[string]interface{}); ok {
		formatted.WriteString("🧠 Memory Usage:\n")
		if total, ok := memInfo["total"].(uint64); ok {
			formatted.WriteString(fmt.Sprintf("  Total: %.2f GB\n", float64(total)/1024/1024/1024))
		}
		if used, ok := memInfo["used"].(uint64); ok {
			formatted.WriteString(fmt.Sprintf("  Used:  %.2f GB\n", float64(used)/1024/1024/1024))
		}
		if free, ok := memInfo["free"].(uint64); ok {
			formatted.WriteString(fmt.Sprintf("  Free:  %.2f GB\n", float64(free)/1024/1024/1024))
		}
		if usedPercent, ok := memInfo["used_percent"].(float64); ok {
			if usedPercent > 80 {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%% (HIGH)\n", usedPercent))
			} else if usedPercent > 50 {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%% (MEDIUM)\n", usedPercent))
			} else {
				formatted.WriteString(fmt.Sprintf("  Usage: %.2f%%\n", usedPercent))
			}
		}
		formatted.WriteString("\n")
	}

	// Disk info
	if diskInfo, ok := input["disk"].(map[string]interface{}); ok {
		formatted.WriteString("💽 Disk Usage:\n")
		for mount, usage := range diskInfo {
			// Skip snap mounts to reduce spam
			if strings.HasPrefix(mount, "/snap") {
				continue
			}

			if u, ok := usage.(map[string]interface{}); ok {
				formatted.WriteString(fmt.Sprintf("  %s:\n", mount))

				usedPercent, hasPercent := u["used_percent"].(float64)
				if hasPercent {
					// Display warning/critical status for high disk usage
					if usedPercent >= 90 {
						formatted.WriteString(fmt.Sprintf("    Usage: %.2f%% 🔴 CRITICAL\n", usedPercent))
						hasErrors = true
					} else if usedPercent >= 80 {
						formatted.WriteString(fmt.Sprintf("    Usage: %.2f%% 🟠 WARNING\n", usedPercent))
						hasErrors = true
					} else {
						formatted.WriteString(fmt.Sprintf("    Usage: %.2f%%\n", usedPercent))
					}
				}

				if total, ok := u["total"].(uint64); ok {
					formatted.WriteString(fmt.Sprintf("    Total: %.2f GB\n", float64(total)/1024/1024/1024))
				}
				if free, ok := u["free"].(uint64); ok {
					formatted.WriteString(fmt.Sprintf("    Free:  %.2f GB\n", float64(free)/1024/1024/1024))
				}
			}
		}
	}

	// Summary
	formatted.WriteString("\n")
	if hasErrors {
		formatted.WriteString("⚠️ Issues detected! Please check the output above.\n")
	} else {
		formatted.WriteString("✅ All systems operational!\n")
	}

	result := formatted.String()

	// info ==> shared
	(*shared)["formatted_output"] = result

	return result, nil
}

func (f *FormatterPlugin) FormatResult(result interface{}) (string, error) {
	if msg, ok := result.(string); ok {
		return msg, nil
	}
	return "", fmt.Errorf("unexpected result type: %T", result)
}

var PluginInstance pluginconf.Plugin = &FormatterPlugin{}
