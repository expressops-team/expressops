// ignore this file, it's just for testing flow execution
package main

import (
	"context"
	"fmt"
	"net/http"

	pluginconf "expressops/internal/plugin/loader" // IMPORTANT: this is the package that defines the Plugin interface

	"github.com/sirupsen/logrus"
)

type TestPrintPlugin struct {
	logger  *logrus.Logger
	message string
}

func (p *TestPrintPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.message = "Hello, World!" // default message

	if msg, ok := config["message"].(string); ok {
		p.message = msg
	}

	p.logger.Info("Inicializando TestPrint Plugin")
	return nil
}

func (p *TestPrintPlugin) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	// Log request information if available
	if r != nil {
		p.logger.Infof("Request recibido desde: %s, User-Agent: %s", r.RemoteAddr, r.UserAgent())
	}

	// Check for message override in shared context
	if shared != nil {
		if msg, ok := (*shared)["message"].(string); ok && msg != "" {
			return msg, nil
		}

		// Check for namespace parameter
		if namespace, ok := (*shared)["namespace"].(string); ok {
			return fmt.Sprintf("Hello from namespace: %s", namespace), nil
		}
	}

	return p.message, nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	return fmt.Sprintf("📝 %v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
