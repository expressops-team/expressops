// ignore this file, it's just for testing flow execution
// :D
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
	p.logger.WithFields(logrus.Fields{
		"pluginName": "TestPrintPlugin",
		"action":     "Initialize",
	}).Info("TestPrintPlugin inicializado")
	return nil
}

func (p *TestPrintPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	pluginName := "TestPrintPlugin"
	flowName := request.URL.Query().Get("flowName")

	result := fmt.Sprintf("👋 Hello, I am a %s plugin running in flow: %s!", pluginName, flowName)
	metrics.IncTestPrint("success")
	return result, nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	p.logger.WithFields(logrus.Fields{
		"pluginName": "TestPrintPlugin",
		"action":     "FormatResult",
	}).Debug("Formateando resultado de TestPrintPlugin")
	if str, ok := result.(string); ok {
		return str, nil
	}
	metrics.IncTestPrint("error_format")
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
