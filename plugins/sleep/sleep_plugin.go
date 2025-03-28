package main

import (
	"context"
	pluginconf "expressops/internal/plugin/loader"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type SleepPlugin struct {
	logger *logrus.Logger
}

func (p *SleepPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Inicializando Sleep Plugin")
	return nil
}

func (p *SleepPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	duration := 10 * time.Second // Dormimos 10 segundos
	p.logger.Info("Sleep Plugin comienza a dormir")

	select {
	case <-time.After(duration):
		p.logger.Info("Sleep Plugin terminó exitosamente")
		return "He terminado de dormir!", nil
	case <-ctx.Done():
		p.logger.Warn("Sleep Plugin ha sido cancelado!")
		return nil, ctx.Err()
	}
}

func (p *SleepPlugin) FormatResult(result interface{}) (string, error) {
	return fmt.Sprintf("Resultado del Sleep Plugin: %v", result), nil
}

var PluginInstance pluginconf.Plugin = &SleepPlugin{}
