// ignore this file, it's just for testing flow execution
package main

import (
	"context"
	"fmt"
	"net/http"

	"expressops/internal/metrics"
	pluginconf "expressops/internal/plugin/loader"

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

	result := fmt.Sprintf("👋 Hello, I am a  %s!", "test")
	metrics.IncTestPrint("success")
	return result, nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	if str, ok := result.(string); ok {
		return str, nil
	}
	metrics.IncTestPrint("error_format")
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
