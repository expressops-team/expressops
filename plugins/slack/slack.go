// plugins/slack/slack.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	pluginconf "expressops/internal/plugin/loader"
	"fmt"
	"net/http"

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
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	message, _ := params["message"].(string)
	channel, _ := params["channel"].(string)
	severity, _ := params["severity"].(string)

	// Prepare the JSON payload
	payload := map[string]interface{}{
		"text":    fmt.Sprintf("[%s] %s", severity, message),
		"channel": channel,
	}
	// Encode the payload to JSON
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(s.webhook, "application/json", bytes.NewBuffer(jsonData))
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
	return "success", nil
}

// PluginInstance is the instance of the plugin that will be registered in the plugin manager at runtime
// so it can be used by the main application
var PluginInstance pluginconf.Plugin = &SlackPlugin{}
