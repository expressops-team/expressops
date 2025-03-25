package main

import (
	"fmt"
	"net/http"
	"sync"

	pluginconf "expressops/internal/plugin/loader"

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
func (p *HealthCheckPlugin) Initialize(config map[string]interface{}) error {
	p.logger.Info("Initializing Health Check Plugin")

	// Example: Add a simple check (memory, etc.)
	p.RegisterCheck("example", func() error {
		return nil // Simulate that it is OK
	})

	return nil
}

// Execute performs the health checks
func (p *HealthCheckPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	p.logger.Info("Performing health check")

	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]string)

	for name, check := range p.checks {
		p.logger.Infof("Running check: %s", name)
		if err := check(); err != nil {
			p.logger.Errorf("Check failed: %s - %v", name, err)
			result[name] = fmt.Sprintf("FAIL: %v", err)
		} else {
			result[name] = "OK"
		}
	}

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
	result, err := p.Execute(nil)
	if err != nil {
		p.logger.Errorf("Health check execution failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Health check failed: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	for k, v := range result.(map[string]string) {
		fmt.Fprintf(w, "%s: %s\n", k, v)
	}
	p.logger.Info("Health check response sent")
}

// Export the plugin instance
var PluginInstance pluginconf.Plugin = NewHealthCheckPlugin(logrus.New())
