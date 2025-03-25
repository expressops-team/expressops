package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	//library for system health checks
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/sirupsen/logrus"
)

type HealthCheckPlugin struct {
	logger *logrus.Logger
	checks map[string]func() error
	mu     sync.Mutex
}

// NewHealthCheckPlugin creates a new instance of HealthCheckPlugin
func NewHealthCheckPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &HealthCheckPlugin{
		logger: logger,
		checks: make(map[string]func() error),
	}
}

// initializes the plugin with the provided configuration
func (p *HealthCheckPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Initializing Health Check Plugin")

	// Register system health checks
	p.RegisterCheck("cpu", p.checkCPU)
	p.RegisterCheck("memory", p.checkMemory)
	p.RegisterCheck("disk", p.checkDisk)

	return nil
}

// checkCPU verifies CPU usage
func (p *HealthCheckPlugin) checkCPU() error {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("error getting CPU usage: %v", err)
	}
	if len(percent) > 0 && percent[0] > 90 {
		return fmt.Errorf("high CPU usage: %.2f%%", percent[0])
	}
	return nil
}

// checkMemory verifies memory usage
func (p *HealthCheckPlugin) checkMemory() error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting memory info: %v", err)
	}
	if v.UsedPercent > 90 {
		return fmt.Errorf("high memory usage: %.2f%%", v.UsedPercent)
	}
	return nil
}

// checkDisk verifies disk space
func (p *HealthCheckPlugin) checkDisk() error {
	parts, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("error getting disk info: %v", err)
	}
	for _, part := range parts {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			continue
		}
		if usage.UsedPercent > 90 {
			return fmt.Errorf("high disk usage on %s: %.2f%%", part.Mountpoint, usage.UsedPercent)
		}
	}
	return nil
}

// Execute performs the health checks
func (p *HealthCheckPlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	p.logger.Info("Performing health check")

	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})

	// Get CPU info
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuPercent[0],
		}
	}

	// Get Memory info
	if memInfo, err := mem.VirtualMemory(); err == nil {
		result["memory"] = map[string]interface{}{
			"total":        memInfo.Total,
			"used":         memInfo.Used,
			"free":         memInfo.Free,
			"used_percent": memInfo.UsedPercent,
		}
	}

	// Get Disk info
	if parts, err := disk.Partitions(false); err == nil {
		diskInfo := make(map[string]interface{})
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
	}

	// Run health checks
	healthStatus := make(map[string]string)
	for name, check := range p.checks {
		p.logger.Infof("Running check: %s", name)
		if err := check(); err != nil {
			p.logger.Errorf("Check failed: %s - %v", name, err)
			healthStatus[name] = fmt.Sprintf("FAIL: %v", err)
		} else {
			healthStatus[name] = "OK"
		}
	}
	result["health_status"] = healthStatus

	p.logger.Info("Health check completed")
	return result, nil
}

// RegisterCheck registers a new custom check
func (p *HealthCheckPlugin) RegisterCheck(name string, check func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checks[name] = check
	p.logger.Infof("Registered health check: %s", name)
}

// Handler for the /healthz endpoint
func (p *HealthCheckPlugin) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	p.logger.Info("Handling /healthz request")
	result, err := p.Execute(r.Context(), nil)
	if err != nil {
		p.logger.Errorf("Health check execution failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Health check failed: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	// Format the output nicely
	fmt.Fprintf(w, "System Health Status:\n\n")

	// Convert result to map[string]interface{}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		fmt.Fprintf(w, "Error: unexpected result type\n")
		return
	}

	// Print health status
	if status, ok := resultMap["health_status"].(map[string]string); ok {
		fmt.Fprintf(w, "Health Checks:\n")
		for k, v := range status {
			fmt.Fprintf(w, "  %s: %s\n", k, v)
		}
		fmt.Fprintf(w, "\n")
	}

	// Print CPU info
	if cpuInfo, ok := resultMap["cpu"].(map[string]interface{}); ok {
		fmt.Fprintf(w, "CPU Usage:\n")
		if usage, ok := cpuInfo["usage_percent"].(float64); ok {
			fmt.Fprintf(w, "  Usage: %.2f%%\n", usage)
		}
		fmt.Fprintf(w, "\n")
	}

	// Print Memory info
	if memInfo, ok := resultMap["memory"].(map[string]interface{}); ok {
		fmt.Fprintf(w, "Memory Usage:\n")
		if total, ok := memInfo["total"].(uint64); ok {
			fmt.Fprintf(w, "  Total: %.2f GB\n", float64(total)/1024/1024/1024)
		}
		if used, ok := memInfo["used"].(uint64); ok {
			fmt.Fprintf(w, "  Used:  %.2f GB\n", float64(used)/1024/1024/1024)
		}
		if free, ok := memInfo["free"].(uint64); ok {
			fmt.Fprintf(w, "  Free:  %.2f GB\n", float64(free)/1024/1024/1024)
		}
		if usedPercent, ok := memInfo["used_percent"].(float64); ok {
			fmt.Fprintf(w, "  Usage: %.2f%%\n", usedPercent)
		}
		fmt.Fprintf(w, "\n")
	}

	// Print Disk info
	if diskInfo, ok := resultMap["disk"].(map[string]interface{}); ok {
		fmt.Fprintf(w, "Disk Usage:\n")
		for mount, usage := range diskInfo {
			if u, ok := usage.(map[string]interface{}); ok {
				fmt.Fprintf(w, "  %s:\n", mount)
				if total, ok := u["total"].(uint64); ok {
					fmt.Fprintf(w, "    Total: %.2f GB\n", float64(total)/1024/1024/1024)
				}
				if used, ok := u["used"].(uint64); ok {
					fmt.Fprintf(w, "    Used:  %.2f GB\n", float64(used)/1024/1024/1024)
				}
				if free, ok := u["free"].(uint64); ok {
					fmt.Fprintf(w, "    Free:  %.2f GB\n", float64(free)/1024/1024/1024)
				}
				if usedPercent, ok := u["used_percent"].(float64); ok {
					fmt.Fprintf(w, "    Usage: %.2f%%\n", usedPercent)
				}
			}
		}
	}

	p.logger.Info("Health check response sent")
}

// Export the plugin instance
var PluginInstance pluginconf.Plugin = NewHealthCheckPlugin(logrus.New())
