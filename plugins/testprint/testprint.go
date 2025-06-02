// ignore this file, it's just for testing flow execution
// :D
package main

import (
	"context"
	"fmt"
	"net/http"

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

	logFields := logrus.Fields{
		"pluginName": pluginName,
		"action":     "Execute",
		"flowName":   flowName,
		"remoteAddr": request.RemoteAddr,
		"userAgent":  request.UserAgent(),
	}
	p.logger.WithFields(logFields).Info("Petición recibida para TestPrintPlugin")

	// Loguear datos de 'shared' si existen, para depuración
	if shared != nil && len(*shared) > 0 {
		p.logger.WithFields(logFields).WithField("sharedData", *shared).Debug("Contenido de 'shared' data")
	}

	p.logger.WithFields(logFields).Info("TestPrintPlugin ejecutado exitosamente")
	return "👋 Hello!  If you see this, the project is working! ✨ :D", nil
}

func (p *TestPrintPlugin) FormatResult(result interface{}) (string, error) {
	p.logger.WithFields(logrus.Fields{
		"pluginName": "TestPrintPlugin",
		"action":     "FormatResult",
	}).Debug("Formateando resultado de TestPrintPlugin")
	if str, ok := result.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TestPrintPlugin{}
