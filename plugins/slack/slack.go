// plugins/slack/slack.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	pluginconf "expressops/internal/plugin/loader"

	"expressops/internal/metrics"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type SlackPlugin struct {
	webhook string
	logger  *logrus.Logger
}

// Initialize initializes the plugin with the provided configuration
func (s *SlackPlugin) Initialize(_ context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	s.logger = logger
	pluginName := "SlackPlugin" // Definir nombre para logs

	webhook, ok := config["webhook_url"].(string)
	if !ok || webhook == "" {
		err := fmt.Errorf("slack webhook URL required")
		s.logger.WithFields(logrus.Fields{
			"pluginName": pluginName,
			"action":     "InitializeFail",
			"error":      err.Error(),
		}).Error("Configuración requerida faltante")
		return err
	}
	s.webhook = webhook

	s.logger.WithFields(logrus.Fields{
		"pluginName":        pluginName,
		"action":            "Initialize",
		"webhookConfigured": true, // No loguear la URL completa por seguridad
	}).Info("SlackPlugin inicializado")
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	pluginName := "SlackPlugin"
	flowName := request.URL.Query().Get("flowName") // Obtener flowName si está disponible

	logFields := logrus.Fields{
		"pluginName": pluginName,
		"flowName":   flowName,
	}

	var message string

	// Try to get message from shared context
	if msgVal, ok := (*shared)["message"]; ok {
		if msgStr, ok := msgVal.(string); ok {
			s.logger.WithFields(logFields).WithField("source", "shared.message").Debugf("Mensaje obtenido: %.50s...", msgStr)
			message = msgStr
		} else {
			s.logger.WithFields(logFields).WithField("sourceType", fmt.Sprintf("%T", msgVal)).Warn("shared[\"message\"] existe pero no es string")
		}
	}

	if message == "" {
		s.logger.WithFields(logFields).Debug("shared[\"message\"] no encontrado o vacío, verificando shared[\"previous_result\"]")
		if prevResult, ok := (*shared)["previous_result"]; ok {
			if prevStr, ok := prevResult.(string); ok {
				s.logger.WithFields(logFields).WithField("source", "shared.previous_result_string").Debugf("Mensaje obtenido: %.50s...", prevStr)
				message = prevStr
			} else {
				message = fmt.Sprintf("%v", prevResult) // Convertir a string
				s.logger.WithFields(logFields).WithField("source", "shared.previous_result_converted").Debugf("Mensaje obtenido: %.50s...", message)
			}
		}
	}

	if message == "" {
		err := fmt.Errorf("no hay mensaje para enviar a Slack")
		s.logger.WithFields(logFields).WithField("error", err.Error()).Error("Mensaje vacío")
		return nil, err
	}

	var channel string
	var channelLabel string
	if channelVal, ok := (*shared)["channel"]; ok {
		if channelStr, ok := channelVal.(string); ok {
			channel = channelStr

			channelLabel = channelStr
		} else {
			s.logger.Warnf("shared[\"channel\"] exists but is not a string (type: %T), using default webhook channel", channelVal)
			channelLabel = "default"
		}
	} else {
		s.logger.Debug("shared[\"channel\"] not found, using default webhook channel")
		channelLabel = "default"

	}

	severity := "info" // Default severity
	if severityVal, ok := (*shared)["severity"]; ok {
		if severityStr, ok := severityVal.(string); ok {
			severity = severityStr
		} else {
			s.logger.WithFields(logFields).WithField("severityType", fmt.Sprintf("%T", severityVal)).Warnf("shared[\"severity\"] no es string, se usará '%s'", severity)
		}
	}
	logFields["severity"] = severity

	s.logger.WithFields(logFields).WithField("messageLength", len(message)).Info("Intentando enviar notificación a Slack")

	payload := map[string]interface{}{"text": message}
	if channel != "" {
		payload["channel"] = channel
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.WithFields(logFields).WithFields(logrus.Fields{
			"action": "ExecuteFail",
			"step":   "MarshalPayload",
			"error":  err.Error(),
		}).Error("Error codificando payload JSON para Slack")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhook, bytes.NewBuffer(payloadBytes))
	if err != nil {
		s.logger.WithFields(logFields).WithFields(logrus.Fields{
			"action": "ExecuteFail",
			"step":   "CreateRequest",
			"error":  err.Error(),
		}).Error("Error creando petición para Slack")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	// ========= Determine status and record metrics ========
	statusLabel := "success"

	if err != nil {

		statusLabel = "error_request"
		s.logger.Errorf("Error sending message to Slack: %v", err)
		metrics.IncSlackNotification(statusLabel, channelLabel)

		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.WithError(err).Error("Error closing response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		statusLabel = "error_api"
		bodyBytes, _ := io.ReadAll(resp.Body)

		s.logger.Errorf("Error in Slack API response: %s - Body: %s", resp.Status, string(bodyBytes))
		metrics.IncSlackNotification(statusLabel, channelLabel)
		return nil, fmt.Errorf("slack API error: %s", resp.Status)
	}

	metrics.IncSlackNotification(statusLabel, channelLabel)

	return "Message sent to Slack", nil
}

// FormatResult follows the original implementation
func (s *SlackPlugin) FormatResult(result interface{}) (string, error) {
	s.logger.WithFields(logrus.Fields{
		"pluginName": "SlackPlugin",
		"action":     "FormatResult",
	}).Debug("Formateando resultado")
	if resultMap, ok := result.(map[string]interface{}); ok {
		return fmt.Sprintf("Slack Result: Status=%v", resultMap["status"]), nil
	}
	return fmt.Sprintf("Slack Result: %v", result), nil
}

// PluginInstance follows the original implementation
var PluginInstance pluginconf.Plugin = &SlackPlugin{}
