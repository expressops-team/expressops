// ignore this file, it's just for testing flow execution
package main

import (
	"context"
	"fmt"

	pluginconf "expressops/internal/plugin/loader" // IMPORTANT: this is the package that defines the Plugin interface

	"github.com/sirupsen/logrus"
)

type TestPrintPlugin struct {
	logger *logrus.Logger
}

func (p *TestPrintPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Inicializando TestPrint Plugin")
	return nil
}

func (p *TestPrintPlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return "Hello, World!", nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
