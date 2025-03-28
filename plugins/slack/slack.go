// plugins/slack/slack.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type SlackPlugin struct {
	webhook string
	logger  *logrus.Logger
}

// Initialize initializes the plugin with the provided configuration
func (s *SlackPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	s.logger = logger
	webhook, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("slack webhook URL required")
	}
	s.webhook = webhook
	s.logger.Info("Inicializando Slack Plugin")
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	var message, channel, severity string

	// Get parameters from shared context
	if shared != nil {
		if msg, ok := (*shared)["message"].(string); ok {
			message = msg
		}
		if ch, ok := (*shared)["channel"].(string); ok {
			channel = ch
		}
		if sev, ok := (*shared)["severity"].(string); ok {
			severity = sev
		}
	}

	// Extract parameters from HTTP request if available
	if r != nil {
		s.logger.Infof("Solicitud de Slack desde: %s", r.RemoteAddr)

		// Check query parameters
		if queryMsg := r.URL.Query().Get("message"); queryMsg != "" {
			message = queryMsg
		}
		if queryCh := r.URL.Query().Get("channel"); queryCh != "" {
			channel = queryCh
		}
		if querySev := r.URL.Query().Get("severity"); querySev != "" {
			severity = querySev
		}
	}

	// Verify required parameters
	if message == "" {
		return nil, fmt.Errorf("message parameter is required")
	}

	// Set defaults if not provided
	if channel == "" {
		channel = "#general"
	}
	if severity == "" {
		severity = "info"
	}

	// Prepare the JSON payload
	payload := map[string]interface{}{
		"text":    fmt.Sprintf("[%s] %s", severity, message),
		"channel": channel,
	}

	// Encode the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error encoding JSON payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.webhook, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logger.Errorf("Error sending message to Slack: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Errorf("Slack API error: %s", resp.Status)
		return nil, fmt.Errorf("slack API error: %s", resp.Status)
	}

	s.logger.Info("Message successfully sent to Slack")
	return map[string]interface{}{
		"status":   "sent",
		"message":  message,
		"channel":  channel,
		"severity": severity,
	}, nil
}

func (s *SlackPlugin) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if status, ok := resultMap["status"].(string); ok && status == "sent" {
			channel := resultMap["channel"].(string)
			severity := resultMap["severity"].(string)
			return fmt.Sprintf("📢 Mensaje enviado a %s con severidad %s", channel, severity), nil
		}
	}
	return "Mensaje enviado a Slack", nil
}

// PluginInstance is the instance of the plugin that will be registered in the plugin manager at runtime
// so it can be used by the main application
var PluginInstance pluginconf.Plugin = &SlackPlugin{}
