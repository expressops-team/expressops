package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
		baseDir = "logs"
	}
	p.baseDir = baseDir

	if err := os.MkdirAll(p.baseDir, 0755); err != nil {
		return fmt.Errorf("could not create logs directory: %v", err)
	}

	p.logger.Info("Initializing Log File Creator Plugin")
	return nil
}

func (p *LogFileCreator) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	var results interface{}
	var flowName string
	var logText []string

	if r != nil {
		p.logger.Infof("Log request from: %s", r.RemoteAddr)
	}

	currentTime := time.Now()
	dateStr := currentTime.Format("20060102")
	timeStr := currentTime.Format("150405")

	dateDir := filepath.Join(p.baseDir, dateStr)
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create date directory: %v", err)
	}

	dailyLogPath := filepath.Join(dateDir, "daily.log")

	if r != nil && r.URL != nil {
		flowName = r.URL.Query().Get("flowName")
	}

	if shared != nil {
		if res, exists := (*shared)["results"]; exists {
			results = res
		}

		if flowName == "" {
			if fn, exists := (*shared)["flow_name"].(string); exists && fn != "" {
				flowName = fn
			}
		}

		if formattedOutput, exists := (*shared)["formatted_output"].(string); exists && formattedOutput != "" {
			logText = append(logText, fmt.Sprintf("===== Formatted output from %s at %s =====",
				flowName, currentTime.Format("2006-01-02 15:04:05")))
			logText = append(logText, formattedOutput)
		}
	}

	if flowName == "" {
		flowName = "unknown"
	}

	logEntry := fmt.Sprintf("time=\"%s\" level=info msg=\"===== Entrada de registro en %s =====\"\n",
		currentTime.Format("2006-01-02 15:04:05"), currentTime.Format("2006-01-02 15:04:05"))
	logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Flow ejecutado: %s\"\n",
		currentTime.Format("2006-01-02 15:04:05"), flowName)

	if shared != nil {
		paramSummary := "  "
		paramCount := 0
		for k, v := range *shared {
			if !strings.HasPrefix(k, "_") && k != "results" && k != "formatted_output" {
				if paramCount > 0 {
					paramSummary += ", "
				}
				paramSummary += fmt.Sprintf("%s: %v", k, v)
				paramCount++
			}
		}

		if paramCount > 0 {
			logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Parámetros del flujo: %s\"\n",
				currentTime.Format("2006-01-02 15:04:05"), paramSummary)
		}
	}

	pluginLogs := make(map[string]string)

	if results != nil {
		resultArray, isArray := results.([]interface{})
		if isArray {
			pluginList := ""
			for _, res := range resultArray {
				if resMap, ok := res.(map[string]interface{}); ok {
					if plugin, ok := resMap["plugin"].(string); ok {
						if pluginList != "" {
							pluginList += ", "
						}
						pluginList += plugin
					}
				}
			}

			if pluginList != "" {
				logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugins ejecutados: %s\"\n",
					currentTime.Format("2006-01-02 15:04:05"), pluginList)
			}

			for idx, res := range resultArray {
				if resMap, ok := res.(map[string]interface{}); ok {
					plugin, hasPlugin := resMap["plugin"].(string)
					if !hasPlugin {
						continue
					}

					var pluginLog strings.Builder

					pluginLog.WriteString(fmt.Sprintf("===== Plugin %s execution at %s =====\n\n",
						plugin, currentTime.Format("2006-01-02 15:04:05")))

					pluginLog.WriteString(fmt.Sprintf("Flow: %s\n", flowName))
					pluginLog.WriteString(fmt.Sprintf("Execution order: %d of %d\n\n", idx+1, len(resultArray)))

					if _, hasError := resMap["error"]; hasError {
						errMsg, _ := resMap["error"].(string)
						pluginLog.WriteString(fmt.Sprintf("Status: ERROR\n"))
						pluginLog.WriteString(fmt.Sprintf("Error message: %s\n\n", errMsg))
					} else {
						pluginLog.WriteString("Status: SUCCESS\n\n")
					}

					if formatted, ok := resMap["formatted_result"].(string); ok && formatted != "" {
						pluginLog.WriteString("Output:\n")
						pluginLog.WriteString(formatted)
						pluginLog.WriteString("\n")
					} else if rawResult, ok := resMap["resultado"]; ok {
						pluginLog.WriteString("Raw output:\n")
						pluginLog.WriteString(fmt.Sprintf("%v\n", rawResult))
					}

					pluginLogs[plugin] = pluginLog.String()

					if _, hasError := resMap["error"]; hasError {
						logEntry += fmt.Sprintf("time=\"%s\" level=error msg=\"Plugin %s ejecutado con errores\"\n",
							currentTime.Format("2006-01-02 15:04:05"), plugin)
					} else {
						if strings.Contains(plugin, "health") {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Revisión de salud del sistema completada\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						} else if strings.Contains(plugin, "format") {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Formateo de resultados completado\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						} else if strings.Contains(plugin, "print") {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Impresión de resultados completada\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						} else if strings.Contains(plugin, "log") {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Registro de resultados completado\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						} else if strings.Contains(plugin, "slack") {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Notificación enviada\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						} else {
							logEntry += fmt.Sprintf("time=\"%s\" level=info msg=\"Plugin ejecutado: %s - Completado\"\n",
								currentTime.Format("2006-01-02 15:04:05"), plugin)
						}
					}
				}
			}
		}
	}

	var file *os.File
	var err error

	if _, err = os.Stat(dailyLogPath); os.IsNotExist(err) {
		file, err = os.Create(dailyLogPath)
	} else {
		file, err = os.OpenFile(dailyLogPath, os.O_APPEND|os.O_WRONLY, 0644)
	}

	if err != nil {
		p.logger.Errorf("Error opening daily log file: %v", err)
		return nil, fmt.Errorf("error opening daily log file: %v", err)
	}
	defer file.Close()

	if _, err = file.WriteString(logEntry); err != nil {
		p.logger.Errorf("Error writing to daily log file: %v", err)
		return nil, fmt.Errorf("error writing to daily log file: %v", err)
	}

	for plugin, pluginLog := range pluginLogs {
		safePluginName := strings.ReplaceAll(plugin, "/", "-")
		safePluginName = strings.ReplaceAll(safePluginName, " ", "_")

		pluginLogFilename := fmt.Sprintf("%s-%s.log", safePluginName, timeStr)
		pluginLogPath := filepath.Join(dateDir, pluginLogFilename)

		if err := os.WriteFile(pluginLogPath, []byte(pluginLog), 0644); err != nil {
			p.logger.Warnf("Could not write log for plugin %s: %v", plugin, err)
		} else {
			p.logger.Infof("Created log for plugin %s at %s", plugin, pluginLogPath)
		}
	}

	if len(logText) > 0 {
		flowLogFilename := fmt.Sprintf("%s-%s.log", flowName, timeStr)
		flowLogPath := filepath.Join(dateDir, flowLogFilename)

		err = os.WriteFile(flowLogPath, []byte(strings.Join(logText, "\n")), 0644)
		if err != nil {
			p.logger.Warnf("Could not write flow log: %v", err)
		} else {
			p.logger.Infof("Created flow log at %s", flowLogPath)
		}
	}

	p.logger.Infof("Daily log entry appended to: %s", dailyLogPath)

	return map[string]interface{}{
		"date_dir":  dateDir,
		"daily_log": dailyLogPath,
		"timestamp": currentTime.Format("2006-01-02 15:04:05"),
		"flow":      flowName,
	}, nil
}

func (p *LogFileCreator) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if dir, ok := resultMap["date_dir"].(string); ok {
			flow, _ := resultMap["flow"].(string)
			return fmt.Sprintf("📝 Logs for flow '%s' created in directory %s", flow, dir), nil
		}
	}
	return "Logs created successfully", nil
}

var PluginInstance pluginconf.Plugin = &LogFileCreator{}
