package main

import (
	"context"
	"expressops/internal/metrics"
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
	p.logger.Info("Initializing Sleep Plugin")
	return nil
}

func (p *SleepPlugin) Execute(ctx context.Context, req *http.Request, shared *map[string]any) (any, error) {
	p.logger.Info("Sleep Plugin starting to sleep")

	configDuration, ok := (*shared)["duration_seconds"].(int)
	if !ok || configDuration <= 0 {
		configDuration = 10
	}

	duration := time.Duration(configDuration) * time.Second

	select {
	case <-time.After(duration):
		p.logger.Info("Sleep Plugin finished successfully")
		metrics.ObserveSleepDuration(float64(configDuration))
		return fmt.Sprintf("Slept for %.0f seconds", duration.Seconds()), nil
	case <-ctx.Done():
		p.logger.Warn("Sleep Plugin has been canceled!")
		metrics.ObserveSleepDuration(0)
		return nil, ctx.Err()
	}
}

func (p *SleepPlugin) FormatResult(result interface{}) (string, error) {
	return fmt.Sprintf("Sleep Plugin Result: %v", result), nil
}

var PluginInstance pluginconf.Plugin = &SleepPlugin{}
