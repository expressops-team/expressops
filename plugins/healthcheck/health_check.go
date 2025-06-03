package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"expressops/internal/metrics"

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

	// Variables para tracking de estado
	allChecksOK := true
	diskResults := make(map[string]interface{})

	// Get CPU metrics
	var cpuUsagePercent float64
	if cpuPercents, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercents) > 0 {
		cpuUsagePercent = cpuPercents[0]
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuUsagePercent,
		}

		// <---REGISTER CPU USAGE GAUGE --->
		metrics.SetResourceUsage("cpu", "", cpuPercents[0])
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

		// <---REGISTER MEMORY USAGE GAUGE --->
		metrics.SetResourceUsage("memory", "", memInfo.UsedPercent)
	}

	if parts, err := disk.Partitions(false); err == nil {
		diskInfo := make(map[string]interface{})

		maxDiskUsageForGauge := 0.0 // To record the worst record on the gauge
		worstMountPoint := ""

		for _, part := range parts {
			// Only consider actual file system partitions, skip others like /dev/loop, /snap
			if !strings.HasPrefix(part.Device, "/dev/sd") &&
				!strings.HasPrefix(part.Device, "/dev/hd") &&
				!strings.HasPrefix(part.Device, "/dev/nvme") &&
				!strings.HasPrefix(part.Device, "/dev/mapper") &&
				part.Fstype != "fuse.portal" {
				p.logger.Debugf("Skipping non-standard partition: %s (Device: %s, Fstype: %s)", part.Mountpoint, part.Device, part.Fstype)
				continue
			}

			if usage, err := disk.Usage(part.Mountpoint); err == nil {
				diskInfo[part.Mountpoint] = map[string]interface{}{
					"total":        usage.Total,
					"used":         usage.Used,
					"free":         usage.Free,
					"used_percent": usage.UsedPercent,
				}
				// Almacenar datos de disco para uso posterior
				diskResults[part.Mountpoint] = map[string]interface{}{
					"total":        usage.Total,
					"used":         usage.Used,
					"free":         usage.Free,
					"used_percent": usage.UsedPercent,
				}
				// <---UPDATE WORST DISK USAGE --->
				if usage.UsedPercent > maxDiskUsageForGauge {
					maxDiskUsageForGauge = usage.UsedPercent
					worstMountPoint = part.Mountpoint
				}
				// <--- END UPDATE --->
			} else {
				p.logger.Warnf("Could not get disk usage for %s: %v", part.Mountpoint, err)
			}
		}
		result["disk"] = diskInfo
		if worstMountPoint != "" { // Ensure we have a valid mount point
			metrics.SetResourceUsage("disk", worstMountPoint, maxDiskUsageForGauge)
			p.logger.Debugf("Set disk resource usage gauge: Mount='%s', Usage=%.2f%%", worstMountPoint, maxDiskUsageForGauge)
		} else {
			p.logger.Debug("No suitable disk mount point found to report for resource usage gauge.")
		}
	}
	result["disk"] = diskResults

	// Execute checks
	healthStatus := make(map[string]string)

	for name, check := range p.checks {
		p.logger.Infof("Running check: %s", name)
		statusLabel := "ok"
		checkSpecificLogFields := logrus.Fields{
			"pluginName": "HealthCheckPlugin",
			"checkName":  name,
		}

		if err := check(); err != nil {
			statusLabel = "fail"
			p.logger.Warnf("Check failed: %s - %v", name, err) // we use warnf because it's not an error, it's a warning

			healthStatus[name] = fmt.Sprintf("FAIL: %v", err)
			allChecksOK = false
		} else {
			healthStatus[name] = "OK"
			p.logger.WithFields(checkSpecificLogFields).Debug("Chequeo de health OK")
		}
		metrics.IncHealthCheckPerformed(name, statusLabel)
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

	parts, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("error getting disk info: %v", err)
	}
	var highUsageMessages []string
	for _, part := range parts {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			p.logger.Debugf("Skipping disk %s due to error: %v", part.Mountpoint, err)
			continue
		}
		if usage.UsedPercent > threshold {
			highUsageMessages = append(highUsageMessages, fmt.Sprintf("high disk usage on %s: %.2f%% (threshold: %.2f%%)", part.Mountpoint, usage.UsedPercent, threshold))
		}
	}

	// Check the main path
	p.logger.WithFields(logrus.Fields{
		"pluginName": "HealthCheckPlugin",
		"checkName":  "disk_threshold",
		"path":       p.pathToCheck,
	}).WithError(err).Warn("No se pudo obtener uso de disco para el path principal del chequeo")

	if len(highUsageMessages) > 0 {
		return fmt.Errorf("high disk usage detected: %s", strings.Join(highUsageMessages, "; "))
	}
	return nil
}

func (p *HealthCheckPlugin) RegisterCheck(name string, check func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checks[name] = check
}

// simple text output for the health check results
func (p *HealthCheckPlugin) FormatResult(_ interface{}) (string, error) {
	return "Health check completed", nil
}

func NewHealthCheckPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &HealthCheckPlugin{
		logger:     logger,
		checks:     make(map[string]func() error),
		thresholds: make(map[string]float64),
	}
}

var PluginInstance pluginconf.Plugin = NewHealthCheckPlugin(logrus.New())
