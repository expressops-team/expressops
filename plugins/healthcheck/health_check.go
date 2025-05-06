package main

import (
	"context"
	"expressops/internal/server"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"
)

// DefaultThresholds defines default threshold values for checks
var DefaultThresholds = map[string]float64{
	"cpu":    90.0,
	"memory": 90.0,
	"disk":   90.0,
}

type HealthCheckPlugin struct {
	logger     *logrus.Logger
	checks     map[string]func() error
	thresholds map[string]float64
	mu         sync.Mutex
}

func (p *HealthCheckPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Initializing Health Check Plugin")
	p.checks = make(map[string]func() error)

	// Initialize thresholds with defaults
	p.thresholds = make(map[string]float64)
	for k, v := range DefaultThresholds {
		p.thresholds[k] = v
	}

	// Override with config if provided
	if thresholds, ok := config["thresholds"].(map[string]interface{}); ok {
		for metricType, value := range thresholds {
			if threshold, ok := value.(float64); ok {
				p.thresholds[metricType] = threshold
				p.logger.Infof("Set %s threshold to %.2f", metricType, threshold)
			}
		}
	}

	p.RegisterCheck("cpu", p.checkCPU)
	p.RegisterCheck("memory", p.checkMemory)
	p.RegisterCheck("disk", p.checkDisk)

	return nil
}

func (p *HealthCheckPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Performing health check")
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})

	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuPercent[0],
		}

		// <---REGISTER CPU USAGE GAUGE --->
		server.SetResourceUsage("cpu", "", cpuPercent[0])

	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		result["memory"] = map[string]interface{}{
			"total":        memInfo.Total,
			"used":         memInfo.Used,
			"free":         memInfo.Free,
			"used_percent": memInfo.UsedPercent,
		}

		// <---REGISTER MEMORY USAGE GAUGE --->
		server.SetResourceUsage("memory", "", memInfo.UsedPercent)
	}

	if parts, err := disk.Partitions(false); err == nil {
		diskInfo := make(map[string]interface{})

		maxDiskUsageForGauge := 0.0 // To record the worst record on the gauge
		worstMountPoint := ""

		for _, part := range parts {
			if usage, err := disk.Usage(part.Mountpoint); err == nil {
				diskInfo[part.Mountpoint] = map[string]interface{}{
					"total":        usage.Total,
					"used":         usage.Used,
					"free":         usage.Free,
					"used_percent": usage.UsedPercent,
				}
			}
		}
		result["disk"] = diskInfo
		if worstMountPoint != "" {
			server.SetResourceUsage("disk", worstMountPoint, maxDiskUsageForGauge)
		}
	}

	healthStatus := make(map[string]string)
	for name, check := range p.checks {
		p.logger.Infof("Running check: %s", name)
		statusLabel := "ok"
		if err := check(); err != nil {
			statusLabel = "fail"
			p.logger.Warnf("Check failed: %s - %v", name, err) // we use warnf because it's not an error, it's a warning
			healthStatus[name] = fmt.Sprintf("FAIL: %v", err)
		} else {
			healthStatus[name] = "OK"
		}
		server.IncHealthCheckPerformed(name, statusLabel)

	}
	result["health_status"] = healthStatus

	p.logger.Info("Health check completed")
	return result, nil
}

// checkCPU verifies that CPU usage is below the configured threshold
func (p *HealthCheckPlugin) checkCPU() error {
	threshold := p.thresholds["cpu"]
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("error getting CPU usage: %v", err)
	}
	if len(percent) > 0 && percent[0] > threshold {
		return fmt.Errorf("high CPU usage: %.2f%% (threshold: %.2f%%)", percent[0], threshold)
	}
	return nil
}

// checkMemory verifies that memory usage is below the configured threshold
func (p *HealthCheckPlugin) checkMemory() error {
	threshold := p.thresholds["memory"]
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting memory info: %v", err)
	}
	if v.UsedPercent > threshold {
		return fmt.Errorf("high memory usage: %.2f%% (threshold: %.2f%%)", v.UsedPercent, threshold)
	}
	return nil
}

// checkDisk verifies that disk usage is below the configured threshold
func (p *HealthCheckPlugin) checkDisk() error {
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
	if len(highUsageMessages) > 0 {
		return fmt.Errorf(strings.Join(highUsageMessages, "; "))
	}
	return nil
}

// RegisterCheck adds a new health check function to the registry
func (p *HealthCheckPlugin) RegisterCheck(name string, check func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checks[name] = check
}

// simple text output for the health check results
func (p *HealthCheckPlugin) FormatResult(result interface{}) (string, error) {
	return "Health check completed", nil
}

func NewHealthCheckPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &HealthCheckPlugin{
		logger:     logger,
		checks:     make(map[string]func() error),
		thresholds: make(map[string]float64),
	}
}

var PluginInstance = NewHealthCheckPlugin(logrus.New())
