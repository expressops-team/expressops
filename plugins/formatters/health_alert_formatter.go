package main

import (
	"context"
	"fmt"
	"net/http"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type FormatterPlugin struct {
	logger *logrus.Logger
}

func (f *FormatterPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	f.logger = logger
	f.logger.Info("Inicializando Health Formatter Plugin")
	return nil
}

func (f FormatterPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	input, ok := (*shared)["_input"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no se recibió _input válido")
	}

	status, ok := input["health_status"].(map[string]string)
	if !ok {
		return "", fmt.Errorf("resultado sin health_status")
	}

	msg := ""
	for k, v := range status {
		if v != "OK" {
			msg += fmt.Sprintf("🚨 %s: %s\n", k, v)
		}
	}

	if msg == "" {
		(*shared)["message"] = "✅ Todo en orden. No se detectaron problemas de salud del sistema."
		return "", nil
	}

	formatted := fmt.Sprintf("⚠️ Problemas detectados:\n%s", msg)
	(*shared)["message"] = formatted
	return formatted, nil
}
func (f *FormatterPlugin) FormatResult(result interface{}) (string, error) {
	if msg, ok := result.(string); ok {
		if msg == "" {
			return "✅ Todo en orden. No se detectaron problemas de salud del sistema.", nil
		}
		return fmt.Sprintf("📋 Mensaje generado para alerta:\n%s", msg), nil
	}
	return "", fmt.Errorf("resultado inesperado: %v", result)
}

var PluginInstance pluginconf.Plugin = &FormatterPlugin{}
