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
	logger *logrus.Logger
}

func (p *TestPrintPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Initializing TestPrint Plugin")
	return nil
}

func (p *TestPrintPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Request received from: " + request.RemoteAddr + ", User-Agent: " + request.UserAgent())

	// Just return a simple colored message with emoji
	return fmt.Sprintf("\033[1;32m👋 Hello, I am a test: %s!\033[0m", "test"), nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	// Just return the result as is without any transformation
	if str, ok := result.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
