package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type LogFileCreator struct {
	baseDir string
	logger  *logrus.Logger
}

func (p *LogFileCreator) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	baseDir, ok := config["log_dir"].(string)
	if !ok {
		baseDir = "logs" // Default to logs directory
	}
	p.baseDir = baseDir

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(p.baseDir, 0755); err != nil {
		return fmt.Errorf("could not create logs directory: %v", err)
	}

	p.logger.Info("Initializing Log File Creator Plugin")
	return nil
}

func (p *LogFileCreator) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	var results interface{}
	var baseFilename string
	var logEntries []string

	// Log the request
	if r != nil {
		p.logger.Infof("Log request from: %s", r.RemoteAddr)
	}

	// Check for results in shared context
	if shared != nil {
		if res, exists := (*shared)["results"]; exists {
			results = res
		}

		// Check for filename in shared context
		if name, exists := (*shared)["filename"].(string); exists && name != "" {
			baseFilename = name
		}
	}

	// Check for parameters in request if available
	if r != nil {
		if filename := r.URL.Query().Get("filename"); filename != "" {
			baseFilename = filename
		}
	}

	// Generate default filename if not provided
	if baseFilename == "" {
		baseFilename = "log_" + time.Now().Format("20060102_150405")
	}

	// Ensure filename has .log extension
	if !strings.HasSuffix(baseFilename, ".log") {
		baseFilename += ".log"
	}

	// Create full path
	logFilePath := filepath.Join(p.baseDir, baseFilename)

	// Begin collecting log entries
	logEntries = append(logEntries, fmt.Sprintf("Log generated at: %s", time.Now().Format("2006-01-02 15:04:05")))

	// Process results based on type
	if results != nil {
		// Extract health information if it's a plugin result
		if healthChecks := p.extractHealthChecks(results); healthChecks != nil {
			logEntries = append(logEntries, "\n--- Health Check Results ---")
			for name, status := range healthChecks {
				logEntries = append(logEntries, fmt.Sprintf("%s: %s", name, status))
			}
		}

		// Add raw results
		logEntries = append(logEntries, "\n--- Raw Results ---")
		resultsJSON, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			logEntries = append(logEntries, "Error serializing results: "+err.Error())
		} else {
			logEntries = append(logEntries, string(resultsJSON))
		}
	} else {
		logEntries = append(logEntries, "No results provided")
	}

	// Write to log file
	err := os.WriteFile(logFilePath, []byte(strings.Join(logEntries, "\n")), 0644)
	if err != nil {
		p.logger.Errorf("Error writing log file: %v", err)
		return nil, fmt.Errorf("error writing log file: %v", err)
	}

	p.logger.Infof("Log file created: %s", logFilePath)

	return map[string]interface{}{
		"filename": baseFilename,
		"path":     logFilePath,
		"entries":  len(logEntries),
	}, nil
}

// FormatResult formats the result of the execution
func (p *LogFileCreator) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if path, ok := resultMap["path"].(string); ok {
			entries, _ := resultMap["entries"].(int)
			return fmt.Sprintf("📝 Log file created at %s with %d entries", path, entries), nil
		}
	}
	return "Log file created successfully", nil
}

// extractHealthChecks extracts health check information from results
func (p *LogFileCreator) extractHealthChecks(results interface{}) map[string]string {
	healthChecks := make(map[string]string)

	// Try to process as array of plugin results
	if resultsArray, ok := results.([]interface{}); ok {
		for _, result := range resultsArray {
			if resultMap, ok := result.(map[string]interface{}); ok {
				pluginName, _ := resultMap["plugin"].(string)
				formatted, _ := resultMap["formatted"].(string)

				if strings.Contains(pluginName, "health") || strings.Contains(formatted, "health") {
					// For health check plugins, extract status
					if strings.Contains(formatted, "💚") {
						healthChecks[pluginName] = "Healthy"
					} else if strings.Contains(formatted, "❤️") {
						healthChecks[pluginName] = "Critical"
					} else if strings.Contains(formatted, "💛") {
						healthChecks[pluginName] = "Warning"
					}

					// Extract individual checks from Kubernetes health check
					if strings.Contains(pluginName, "kube") {
						p.extractKubeHealthDetails(formatted, healthChecks)
					}
				}
			}
		}
	}

	// If we found any health checks, return them
	if len(healthChecks) > 0 {
		return healthChecks
	}

	return nil
}

// extractKubeHealthDetails extracts details from Kubernetes health check formatted output
func (p *LogFileCreator) extractKubeHealthDetails(formatted string, healthChecks map[string]string) {
	// Extract pod statuses
	podRegex := regexp.MustCompile(`(\w+):\s+(\w+)`)
	matches := podRegex.FindAllStringSubmatch(formatted, -1)

	for _, match := range matches {
		if len(match) == 3 {
			podName := match[1]
			podStatus := match[2]
			healthChecks["pod:"+podName] = podStatus
		}
	}
}

var PluginInstance pluginconf.Plugin = &LogFileCreator{}
