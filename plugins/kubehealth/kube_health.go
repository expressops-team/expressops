// plugins/kubehealth/kube_health.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type KubeHealthPlugin struct {
	logger *logrus.Logger
	config map[string]interface{}
}

// Initialize sets up the plugin
func (p *KubeHealthPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.config = config
	p.logger.Info("Initializing KubeHealth Plugin")
	return nil
}

// Execute runs kubectl and formats the output
func (p *KubeHealthPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Checking Kubernetes pods")

	// Get namespace from config or use default
	namespace := "default"
	if ns, ok := p.config["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Run kubectl command
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
	output, err := cmd.CombinedOutput()

	// Handle command errors
	if err != nil {
		errMsg := fmt.Sprintf("*Kubernetes Error*\n\n```\n%v\n```", err)
		(*shared)["message"] = errMsg
		(*shared)["severity"] = "critical"
		return nil, nil
	}

	// Parse output into pod data
	pods := parsePodOutput(string(output))

	// Handle empty results
	if len(pods) == 0 {
		warnMsg := fmt.Sprintf("*No pods found in namespace '%s'*", namespace)
		(*shared)["message"] = warnMsg
		(*shared)["severity"] = "warning"
		return pods, nil
	}

	// Count problem pods
	problemCount := 0
	for _, pod := range pods {
		if pod["status"] != "Running" {
			problemCount++
		}
	}

	// Format message for Slack
	message := formatSlackMessage(namespace, pods, problemCount)

	// Set shared data for Slack plugin
	(*shared)["message"] = message
	(*shared)["kube_health_results"] = pods

	// Set severity based on ==> problem count <==
	severity := "info"
	if problemCount > 0 {
		severity = "warning"
	}
	if problemCount > 2 {
		severity = "critical"
	}
	(*shared)["severity"] = severity

	return pods, nil
}

// Parse kubectl output into structured pod data
func parsePodOutput(output string) []map[string]string {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return []map[string]string{}
	}

	var pods []map[string]string

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		ready := fields[1]
		status := fields[2]

		age := ""
		restarts := "0"
		if len(fields) > 4 {
			restarts = fields[3]
			age = fields[4]
		}

		emoji := "✅"
		if status != "Running" {
			emoji = "❌"
		} else if !strings.HasPrefix(ready, "1/1") {
			emoji = "⚠️"
			status = "Running (Not Ready)"
		}

		pods = append(pods, map[string]string{
			"name":     name,
			"status":   status,
			"age":      age,
			"ready":    ready,
			"restarts": restarts,
			"emoji":    emoji,
		})
	}

	return pods
}

// Format a nice;) Slack message with the pod status
func formatSlackMessage(namespace string, pods []map[string]string, problemCount int) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("*Kubernetes Pods in `%s`* - %s\n\n",
		namespace, time.Now().Format("2006-01-02 15:04:05")))

	running := []map[string]string{}
	problem := []map[string]string{}

	for _, pod := range pods {
		if pod["status"] == "Running" {
			running = append(running, pod)
		} else {
			problem = append(problem, pod)
		}
	}

	if len(problem) > 0 {
		msg.WriteString("🔴 *Problem Pods:*\n")
		for _, pod := range problem {
			msg.WriteString(fmt.Sprintf("• `%s` - %s %s",
				pod["name"], pod["status"], pod["emoji"]))

			if pod["restarts"] != "0" {
				msg.WriteString(fmt.Sprintf(" (Restarts: %s)", pod["restarts"]))
			}
			msg.WriteString("\n")
		}
		msg.WriteString("\n")
	}

	if len(running) > 0 {
		msg.WriteString("🟢 *Healthy Pods:*\n")
		for _, pod := range running {
			msg.WriteString(fmt.Sprintf("• `%s` %s\n", pod["name"], pod["emoji"]))
		}
		msg.WriteString("\n")
	}

	// Add summary
	totalPods := len(pods)
	msg.WriteString(fmt.Sprintf("*Summary:* %d total pods, %d with issues\n",
		totalPods, problemCount))

	if problemCount > 0 {
		msg.WriteString("\n⚠️ *Action needed!* Check problematic pods.")
	} else {
		msg.WriteString("\n✅ *All pods are healthy!*")
	}

	return msg.String()
}

func (p *KubeHealthPlugin) FormatResult(result interface{}) (string, error) {
	pods, ok := result.([]map[string]string)
	if !ok || len(pods) == 0 {
		return "No pods found", nil
	}

	var sb strings.Builder
	sb.WriteString("Kubernetes Pod Status:\n\n")

	for _, pod := range pods {
		sb.WriteString(fmt.Sprintf("%s: %s %s\n",
			pod["name"], pod["status"], pod["emoji"]))
	}

	return sb.String(), nil
}

// exporting
var PluginInstance pluginconf.Plugin = &KubeHealthPlugin{}
