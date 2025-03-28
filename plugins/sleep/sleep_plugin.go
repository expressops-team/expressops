package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	pluginconf "expressops/internal/plugin/loader"

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

func (p *SleepPlugin) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	// Default sleep duration
	duration := 10 * time.Second

	// Get duration from shared context if available
	if shared != nil {
		if durationParam, ok := (*shared)["duration"]; ok {
			switch v := durationParam.(type) {
			case string:
				// Try to parse as duration string (e.g., "5s", "1m")
				if parsedDuration, err := time.ParseDuration(v); err == nil {
					duration = parsedDuration
				}
			case int:
				duration = time.Duration(v) * time.Second
			case float64:
				duration = time.Duration(v) * time.Second
			}
		}
	}

	// Log request info if available
	if r != nil {
		// Check for duration query parameter
		if durParam := r.URL.Query().Get("duration"); durParam != "" {
			if seconds, err := strconv.Atoi(durParam); err == nil {
				duration = time.Duration(seconds) * time.Second
			}
		}

		p.logger.Infof("Sleep Plugin iniciado desde %s, durmiendo por %v", r.RemoteAddr, duration)
	} else {
		p.logger.Infof("Sleep Plugin comienza a dormir por %v", duration)
	}

	select {
	case <-time.After(duration):
		p.logger.Info("Sleep Plugin terminó exitosamente")
		return map[string]interface{}{
			"status":    "completed",
			"slept_for": duration.String(),
		}, nil
	case <-ctx.Done():
		p.logger.Warn("Sleep Plugin ha sido cancelado!")
		return map[string]interface{}{
			"status":    "cancelled",
			"error":     ctx.Err().Error(),
			"slept_for": "unknown",
		}, ctx.Err()
	}
}

func (p *SleepPlugin) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if status, ok := resultMap["status"].(string); ok {
			if status == "completed" {
				return "😴 He terminado de dormir por " + resultMap["slept_for"].(string), nil
			} else {
				return "⚠️ Sleep cancelado: " + resultMap["error"].(string), nil
			}
		}
	}
	return "Sleep completado", nil
}

var PluginInstance pluginconf.Plugin = &SleepPlugin{}
