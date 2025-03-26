package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	pluginconf "expressops/internal/plugin/loader"

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

func NewHealthCheckPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &HealthCheckPlugin{
		logger: logger,
		checks: make(map[string]func() error),
	}
}

func (p *HealthCheckPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Inicializando Health Check Plugin")
	p.RegisterCheck("cpu", p.checkCPU)
	p.RegisterCheck("memory", p.checkMemory)
	p.RegisterCheck("disk", p.checkDisk)
	return nil
}

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

func (p *HealthCheckPlugin) RegisterCheck(name string, check func() error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checks[name] = check
}

func (p *HealthCheckPlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	p.logger.Info("Performing health check")
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})

	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		result["cpu"] = map[string]interface{}{
			"usage_percent": cpuPercent[0],
		}
	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		result["memory"] = map[string]interface{}{
			"total":        memInfo.Total,
			"used":         memInfo.Used,
			"free":         memInfo.Free,
			"used_percent": memInfo.UsedPercent,
		}
	}

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

	healthStatus := make(map[string]string)
	for name, check := range p.checks {
		p.logger.Infof("Running check: %s", name)
		if err := check(); err != nil {
			p.logger.Warnf("Check failed: %s - %v", name, err) // we use warnf because it's not an error, it's a warning
			healthStatus[name] = fmt.Sprintf("FAIL: %v", err)
		} else {
			healthStatus[name] = "OK"
		}
	}
	result["health_status"] = healthStatus

	p.logger.Info("Health check completed")
	return result, nil
}

func (p *HealthCheckPlugin) FormatResult(result interface{}) (string, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected result type")
	}

	var sb strings.Builder
	sb.WriteString("\n\033[32mSystem Health Status:\033[0m\n\n")

	if status, ok := resultMap["health_status"].(map[string]string); ok {
		sb.WriteString("\033[31mHealth Checks:\033[0m\n")
		for k, v := range status {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
		sb.WriteString("\n")
	}

	if cpuInfo, ok := resultMap["cpu"].(map[string]interface{}); ok {
		sb.WriteString("\033[34mCPU Usage:\033[0m\n")
		if usage, ok := cpuInfo["usage_percent"].(float64); ok {
			sb.WriteString(fmt.Sprintf("  Usage: %.2f%%\n\n", usage))
		}
	}

	if memInfo, ok := resultMap["memory"].(map[string]interface{}); ok {
		sb.WriteString("\033[33mMemory Usage:\033[0m\n")
		if total, ok := memInfo["total"].(uint64); ok {
			sb.WriteString(fmt.Sprintf("  Total: %.2f GB\n", float64(total)/1024/1024/1024))
		}
		if used, ok := memInfo["used"].(uint64); ok {
			sb.WriteString(fmt.Sprintf("  Used:  %.2f GB\n", float64(used)/1024/1024/1024))
		}
		if free, ok := memInfo["free"].(uint64); ok {
			sb.WriteString(fmt.Sprintf("  Free:  %.2f GB\n", float64(free)/1024/1024/1024))
		}
		if usedPercent, ok := memInfo["used_percent"].(float64); ok {
			sb.WriteString(fmt.Sprintf("  Usage: %.2f%%\n\n", usedPercent))
		}
	}

	if diskInfo, ok := resultMap["disk"].(map[string]interface{}); ok {
		sb.WriteString("\033[35mDisk Usage:\033[0m\n")
		for mount, usage := range diskInfo {
			if len(mount) >= 5 && mount[:5] == "/snap" {
				continue
			}
			if u, ok := usage.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf("  %s:\n", mount))
				if total, ok := u["total"].(uint64); ok {
					sb.WriteString(fmt.Sprintf("    Total: %.2f GB\n", float64(total)/1024/1024/1024))
				}
				if used, ok := u["used"].(uint64); ok {
					sb.WriteString(fmt.Sprintf("    Used:  %.2f GB\n", float64(used)/1024/1024/1024))
				}
				if free, ok := u["free"].(uint64); ok {
					sb.WriteString(fmt.Sprintf("    Free:  %.2f GB\n", float64(free)/1024/1024/1024))
				}
				if usedPercent, ok := u["used_percent"].(float64); ok {
					sb.WriteString(fmt.Sprintf("    Usage: %.2f%%\n", usedPercent))
				}
			}
		}
	}

	sb.WriteString("\nDone ✅\n")
	return sb.String(), nil
}

var PluginInstance = NewHealthCheckPlugin(logrus.New())
