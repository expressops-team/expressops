package main

import (
	"context"
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

func (p *HealthCheckPlugin) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	p.logger.Info("Ejecutando Health Check Plugin")

	// Log request info if available
	if r != nil {
		p.logger.Infof("Health check solicitado desde: %s", r.RemoteAddr)
	}

	results := make(map[string]string)

	// Get checks to run from shared context
	checksToRun := []string{}
	if shared != nil {
		if specificChecks, ok := (*shared)["checks"].([]string); ok && len(specificChecks) > 0 {
			checksToRun = specificChecks
		}
	}

	// If no specific checks requested, run all
	if len(checksToRun) == 0 {
		p.mu.Lock()
		for name := range p.checks {
			checksToRun = append(checksToRun, name)
		}
		p.mu.Unlock()
	}

	// Run the checks
	for _, name := range checksToRun {
		p.mu.Lock()
		check, exists := p.checks[name]
		p.mu.Unlock()

		if !exists {
			results[name] = "check not found"
			continue
		}

		if err := check(); err != nil {
			results[name] = fmt.Sprintf("❌ %v", err)
		} else {
			results[name] = "✅ OK"
		}
	}

	return results, nil
}

func (p *HealthCheckPlugin) FormatResult(result interface{}) (string, error) {
	results, ok := result.(map[string]string)
	if !ok {
		return "", fmt.Errorf("unexpected result type: %T", result)
	}

	var sb strings.Builder
	sb.WriteString("💚 Health Check Results:\n")

	for check, status := range results {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", check, status))
	}

	return sb.String(), nil
}

var PluginInstance = NewHealthCheckPlugin(nil)
