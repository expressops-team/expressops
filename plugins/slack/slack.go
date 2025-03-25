// plugins/slack/slack.go
package main

import (
	"bytes"
	"encoding/json"
	pluginconf "expressops/internal/plugin/loader"
	"fmt"
	"net/http"
)

type SlackPlugin struct {
	webhook string
	enabled bool
}

// Initialize initializes the plugin with the provided configuration
func (s *SlackPlugin) Initialize(config map[string]interface{}) error {
	webhook, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("slack webhook URL required")
	}
	s.webhook = webhook
	s.enabled = true
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	if !s.enabled {
		return nil, fmt.Errorf("plugin not initialized")
	}

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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slack API error: %s", resp.Status)
	}

	return "success", nil
}

// PluginInstance is the instance of the plugin that will be registered in the plugin manager at runtime
// so it can be used by the main application
var PluginInstance pluginconf.Plugin = &SlackPlugin{}
