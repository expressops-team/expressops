package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"
)

// Variable to measure start time
var startTime = time.Now()

// DefaultThresholds defines default threshold values for checks
var DefaultThresholds = map[string]float64{
	"cpu":    90.0,
	"memory": 90.0,
	"disk":   90.0,
}

type HealthCheckPlugin struct {
	logger      *logrus.Logger
	checks      map[string]func() error
	thresholds  map[string]float64
	mu          sync.Mutex
	pathToCheck string
}

func (p *HealthCheckPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	pluginName := "HealthCheckPlugin"
	logFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "Initialize",
	}

	p.checks = make(map[string]func() error)
	p.thresholds = make(map[string]float64)
	for k, v := range DefaultThresholds {
		p.thresholds[k] = v
	}

	if thresholdsConf, ok := config["thresholds"].(map[string]interface{}); ok {
		logFields["thresholdsConfigured"] = true
		for metricType, value := range thresholdsConf {
			if threshold, ok := value.(float64); ok {
				p.thresholds[metricType] = threshold
				p.logger.WithFields(logFields).WithFields(logrus.Fields{
					"metricType": metricType,
					"threshold":  threshold,
				}).Debug("Umbral personalizado configurado")
			}
		}
	} else {
		logFields["thresholdsConfigured"] = false
	}

	if path, ok := config["path"].(string); ok {
		p.pathToCheck = path
	} else {
		p.pathToCheck = "/" // Default path
	}
	logFields["pathToCheck"] = p.pathToCheck

	p.logger.WithFields(logFields).Info("HealthCheckPlugin inicializado")

	// Registrar los chequeos que este plugin realizará
	p.RegisterCheck("cpu_threshold", p.checkCPUThreshold)
	p.RegisterCheck("memory_threshold", p.checkMemoryThreshold)
	p.RegisterCheck("disk_threshold", p.checkDiskThreshold)

	return nil
}

func (p *HealthCheckPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	pluginName := "HealthCheckPlugin"
	flowName := request.URL.Query().Get("flowName")
	hostname, _ := os.Hostname()

	logFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "ExecuteStart",
		"flowName":   flowName,
		"hostname":   hostname,
	}
	p.logger.WithFields(logFields).Info("Iniciando health check")

	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})
	result["hostname"] = hostname

	// Get CPU metrics
	var cpuUsagePercent float64
	if cpuPercents, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercents) > 0 {
		cpuUsagePercent = cpuPercents[0]
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuUsagePercent,
		}
	} else {
		p.logger.WithFields(logFields).WithError(err).Error("Error obteniendo porcentaje de CPU")
	}

	// Get memory metrics
	var memoryUsedPercent float64
	if memInfo, err := mem.VirtualMemory(); err == nil {
		memoryUsedPercent = memInfo.UsedPercent
		result["memory"] = map[string]interface{}{
			"total":        memInfo.Total,
			"used":         memInfo.Used,
			"free":         memInfo.Free,
			"used_percent": memInfo.UsedPercent,
		}
	} else {
		p.logger.WithFields(logFields).WithError(err).Error("Error obteniendo información de memoria")
	}

	diskResults := make(map[string]interface{})
	if usage, err := disk.Usage(p.pathToCheck); err == nil {
		diskResults[p.pathToCheck] = map[string]interface{}{
			"total":        usage.Total,
			"used":         usage.Used,
			"free":         usage.Free,
			"used_percent": usage.UsedPercent,
		}
	} else {
		p.logger.WithFields(logFields).WithField("path", p.pathToCheck).WithError(err).Error("Error obteniendo uso de disco para path principal")
	}
	result["disk"] = diskResults

	// Execute checks
	healthStatus := make(map[string]string)
	allChecksOK := true
	for name, checkFunc := range p.checks {
		checkSpecificLogFields := make(logrus.Fields)
		for k, v := range logFields {
			checkSpecificLogFields[k] = v
		}
		checkSpecificLogFields["checkName"] = name

		p.logger.WithFields(checkSpecificLogFields).Debug("Ejecutando chequeo de health")
		if err := checkFunc(); err != nil {
			p.logger.WithFields(checkSpecificLogFields).WithError(err).Warn("Chequeo de health fallido")
			healthStatus[name] = fmt.Sprintf("FAIL: %v", err)
			allChecksOK = false
		} else {
			healthStatus[name] = "OK"
			p.logger.WithFields(checkSpecificLogFields).Debug("Chequeo de health OK")
		}
	}
	result["health_status"] = healthStatus

	result["timestamp"] = time.Now().Unix()
	result["uptime_seconds"] = time.Since(startTime).Seconds()

	finalLogFieldsMap := make(logrus.Fields)
	for k, v := range logFields {
		finalLogFieldsMap[k] = v
	}
	finalLogFieldsMap["action"] = "ExecuteEnd"
	finalLogFieldsMap["allChecksOK"] = allChecksOK
	finalLogFieldsMap["cpuUsage"] = fmt.Sprintf("%.2f%%", cpuUsagePercent)
	finalLogFieldsMap["memoryUsage"] = fmt.Sprintf("%.2f%%", memoryUsedPercent)

	if du, ok := diskResults[p.pathToCheck].(map[string]interface{}); ok {
		if dup, ok := du["used_percent"].(float64); ok {
			finalLogFieldsMap["diskUsagePathMain"] = fmt.Sprintf("%.2f%%", dup)
		}
	}

	if allChecksOK {
		p.logger.WithFields(finalLogFieldsMap).Info("Health check completado exitosamente, todos los chequeos OK")
	} else {
		p.logger.WithFields(finalLogFieldsMap).Warn("Health check completado, algunos chequeos fallaron")
	}

	return result, nil
}

func (p *HealthCheckPlugin) checkCPUThreshold() error {
	threshold := p.thresholds["cpu"]
	percents, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("error obteniendo uso de CPU: %v", err)
	}
	if len(percents) > 0 && percents[0] > threshold {
		return fmt.Errorf("uso de CPU alto: %.2f%% (umbral: %.2f%%)", percents[0], threshold)
	}
	return nil
}

func (p *HealthCheckPlugin) checkMemoryThreshold() error {
	threshold := p.thresholds["memory"]
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error obteniendo información de memoria: %v", err)
	}
	if v.UsedPercent > threshold {
		return fmt.Errorf("uso de memoria alto: %.2f%% (umbral: %.2f%%)", v.UsedPercent, threshold)
	}
	return nil
}

func (p *HealthCheckPlugin) checkDiskThreshold() error {
	threshold := p.thresholds["disk"]
	if usage, err := disk.Usage(p.pathToCheck); err == nil {
		if usage.UsedPercent > threshold {
			return fmt.Errorf("uso de disco alto en %s: %.2f%% (umbral: %.2f%%)", p.pathToCheck, usage.UsedPercent, threshold)
		}
	} else {
		p.logger.WithFields(logrus.Fields{
			"pluginName": "HealthCheckPlugin",
			"checkName":  "disk_threshold",
			"path":       p.pathToCheck,
		}).WithError(err).Warn("No se pudo obtener uso de disco para el path principal del chequeo")
	}
	return nil
}

func (p *HealthCheckPlugin) RegisterCheck(name string, check func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checks[name] = check
}

func (p *HealthCheckPlugin) FormatResult(result interface{}) (string, error) {
	pluginName := "HealthCheckPlugin"
	logFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "FormatResult",
	}
	p.logger.WithFields(logFields).Info("Formateando resultado de health check")

	data, ok := result.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("tipo de resultado inesperado: %T", result)
		p.logger.WithFields(logFields).WithField("error", err.Error()).Error("Error formateando resultado")
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Hostname: %v\n", data["hostname"]))
	if cpuData, ok := data["cpu"].(map[string]interface{}); ok {
		sb.WriteString(fmt.Sprintf("CPU Usage: %.2f%%\n", cpuData["usage_percent"]))
	}
	if memData, ok := data["memory"].(map[string]interface{}); ok {
		sb.WriteString(fmt.Sprintf("Memory Usage: %.2f%% (Used: %.2fGB, Total: %.2fGB)\n",
			memData["used_percent"],
			float64(memData["used"].(uint64))/1024/1024/1024,
			float64(memData["total"].(uint64))/1024/1024/1024))
	}
	if diskData, ok := data["disk"].(map[string]interface{}); ok {
		if mainPathData, ok := diskData[p.pathToCheck].(map[string]interface{}); ok {
			sb.WriteString(fmt.Sprintf("Disk Usage (%s): %.2f%% (Used: %.2fGB, Total: %.2fGB)\n",
				p.pathToCheck,
				mainPathData["used_percent"],
				float64(mainPathData["used"].(uint64))/1024/1024/1024,
				float64(mainPathData["total"].(uint64))/1024/1024/1024))
		}
	}
	if status, ok := data["health_status"].(map[string]string); ok {
		sb.WriteString("Status Checks:\n")
		for name, state := range status {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", name, state))
		}
	}
	return sb.String(), nil
}

func NewHealthCheckPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &HealthCheckPlugin{
		logger:     logger,
		checks:     make(map[string]func() error),
		thresholds: make(map[string]float64),
	}
}

var PluginInstance pluginconf.Plugin = NewHealthCheckPlugin(logrus.New())
