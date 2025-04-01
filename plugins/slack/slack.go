// plugins/slack/slack.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	pluginconf "expressops/internal/plugin/loader"
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
func (s *SlackPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	s.logger = logger
	s.logger.Info("Initializing Slack Plugin")
	webhook, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("slack webhook URL required")
	}
	s.webhook = webhook
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(ctx context.Context, _ *http.Request, shared *map[string]any) (interface{}, error) {
	var message string

	// Try to get message from shared context
	if msgVal, ok := (*shared)["message"]; ok {
		if msgStr, ok := msgVal.(string); ok {
			s.logger.Debugf("Message obtained from shared[\"message\"]: %s", msgStr)
			message = msgStr
		} else {
			s.logger.Warnf("shared[\"message\"] exists but is not a string (type: %T)", msgVal)
		}
	}

	// If no message found, try to get it from previous_result
	if message == "" {
		s.logger.Debug("shared[\"message\"] not found or empty, checking shared[\"previous_result\"]")

		if prevResult, ok := (*shared)["previous_result"]; ok {
			if prevStr, ok := prevResult.(string); ok {
				s.logger.Debugf("Message obtained from shared[\"previous_result\"] (was string): %s", prevStr)
				message = prevStr
			} else {
				// Try to convert to string
				message = fmt.Sprintf("%v", prevResult)
				s.logger.Debugf("Message obtained from shared[\"previous_result\"] (converted to string): %s", message)
			}
		}
	}

	// Check if we have a message to send
	if message == "" {
		err := fmt.Errorf("no message to send")
		s.logger.Error(err.Error())
		return nil, err
	}

	// Get custom channel from shared context (optional)
	var channel string
	if channelVal, ok := (*shared)["channel"]; ok {
		if channelStr, ok := channelVal.(string); ok {
			channel = channelStr
		} else {
			s.logger.Warnf("shared[\"channel\"] exists but is not a string (type: %T), using default webhook channel", channelVal)
		}
	} else {
		s.logger.Debug("shared[\"channel\"] not found, using default webhook channel")
	}

	// Get severity from shared context (optional)
	severity := "info"
	if severityVal, ok := (*shared)["severity"]; ok {
		if severityStr, ok := severityVal.(string); ok {
			severity = severityStr
		} else {
			s.logger.Warnf("shared[\"severity\"] exists but is not a string (type: %T), using '%s'", severityVal, severity)
		}
	} else {
		s.logger.Debugf("shared[\"severity\"] not found, using '%s'", severity)
	}

	s.logger.Infof("Sending to Slack: Channel='%s', Severity='%s', Message='%.50s...'", channel, severity, message) // Log before sending

	// Create payload
	payload := map[string]interface{}{
		"text": message,
	}

	if channel != "" {
		payload["channel"] = channel
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Errorf("Error encoding JSON payload for Slack: %v", err)
		return nil, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhook, bytes.NewBuffer(payloadBytes))
	if err != nil {
		s.logger.Errorf("Error creating request for Slack: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Errorf("Error sending message to Slack: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.logger.Errorf("Error in Slack API response: %s - Body: %s", resp.Status, string(bodyBytes))
		return nil, fmt.Errorf("slack API error: %s", resp.Status)
	}

	s.logger.Info("Message sent successfully to Slack.")
	return "Message sent to Slack", nil
}

// FormatResult follows the original implementation
func (s *SlackPlugin) FormatResult(result interface{}) (string, error) {
	// Podrías hacer este formato más útil si el resultado es el mapa que devolvemos ahora
	if resultMap, ok := result.(map[string]interface{}); ok {
		return fmt.Sprintf("Resultado Slack: Status=%v", resultMap["status"]), nil
	}
	return fmt.Sprintf("Resultado Slack: %v", result), nil
}

// PluginInstance follows the original implementation
var PluginInstance pluginconf.Plugin = &SlackPlugin{}
