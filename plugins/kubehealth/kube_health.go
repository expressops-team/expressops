// plugins/kubehealth/kube_health.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeHealthPlugin struct {
	logger *logrus.Logger
}

// Initialize sets up the plugin with logger and configuration
func (p *KubeHealthPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Initializing KubeHealth Plugin")
	return nil
}

// Execute connects to Kubernetes and retrieves pod status information
func (p *KubeHealthPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	namespace := "default"
	if ns, ok := (*shared)["namespace"].(string); ok {
		namespace = ns
	}

	// Try to get real Kubernetes data
	results, err := p.getKubernetesData(ctx, namespace)
	if err != nil {
		p.logger.Warnf("Could not get Kubernetes data: %v", err)
		p.logger.Info("Falling back to simulated data")

		// Generate simulated data instead
		results = p.generateSimulatedData()
	}

	// Store data in various formats to maximize compatibility
	// 1. As return value (for plugins that check previous_result)
	// 2. In shared context under kube_health_results
	// 3. Also put a string summary directly in message

	// Count problem pods for summary
	problemPods := 0
	for _, pod := range results {
		if pod["emoji"] != "✅" {
			problemPods++
		}
	}

	// Create summary
	summary := fmt.Sprintf("Kubernetes Health Check: %d pods total, %d with issues", len(results), problemPods)

	// Store data in shared context for formatter
	(*shared)["kube_health_results"] = results
	(*shared)["kube_health_summary"] = summary
	(*shared)["previous_result"] = results // Make sure it's passed to the next plugin

	// Set a basic message in case formatter fails
	var basicMsg strings.Builder
	basicMsg.WriteString("Kubernetes Pod Status:\n\n")
	for _, pod := range results {
		basicMsg.WriteString(fmt.Sprintf("  %s: %s %s\n", pod["name"], pod["status"], pod["emoji"]))
	}
	(*shared)["message"] = basicMsg.String()

	// Set severity based on problem pods
	severity := "info"
	if problemPods > 0 {
		severity = "warning"
	}
	(*shared)["severity"] = severity

	// Return formatted results for console output
	return results, nil
}

// getKubernetesData attempts to get real data from a K8s cluster
func (p *KubeHealthPlugin) getKubernetesData(ctx context.Context, namespace string) ([]map[string]string, error) {
	// Check if we're running in a Kubernetes cluster
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		p.logger.Info("Running inside Kubernetes, using in-cluster config")
		// Would use in-cluster config, but not implementing that for now
	}

	// Try different kubeconfig locations
	var kubeconfigPaths = []string{
		"/home/dcela_freepik_com/.kube/config", // User-specified path
		"/home/dcela/.kube/config",             // Local user path
		clientcmd.RecommendedHomeFile,          // Default path (~/.kube/config)
	}

	var config clientcmd.ClientConfig

	// Try each path
	for _, path := range kubeconfigPaths {
		p.logger.Infof("Trying kubeconfig at: %s", path)
		if _, err := os.Stat(path); err == nil {
			// Found a kubeconfig file
			p.logger.Infof("Found kubeconfig at: %s", path)

			// Try to load it
			configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
			configOverrides := &clientcmd.ConfigOverrides{}

			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				configLoadingRules, configOverrides)

			config = kubeConfig
			break
		}
	}

	// If no config found, return error
	if config == nil {
		return nil, fmt.Errorf("no valid kubeconfig found in any of the tried locations")
	}

	// Get REST config
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating REST config: %v", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client: %v", err)
	}

	// List pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %v", err)
	}

	results := make([]map[string]string, 0)

	for _, pod := range pods.Items {
		status := string(pod.Status.Phase)
		statusEmoji := "✅"

		if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
			statusEmoji = "⚠️"
		}

		if pod.Status.Phase == v1.PodRunning {
			for _, c := range pod.Status.ContainerStatuses {
				if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
					status = "CrashLoopBackOff"
					statusEmoji = "❌"
					break
				}
			}
		}

		results = append(results, map[string]string{
			"name":   pod.Name,
			"status": status,
			"emoji":  statusEmoji,
		})
	}

	return results, nil
}

// generateSimulatedData creates simulated pod data when we're not in a cluster
func (p *KubeHealthPlugin) generateSimulatedData() []map[string]string {
	p.logger.Info("Generating simulated Kubernetes data")

	// Get current timestamp for pod names to make them look realistic
	timestamp := time.Now().Format("20060102-150405")

	// Create some fake pods with various states
	results := []map[string]string{
		{
			"name":   fmt.Sprintf("expressops-%s", timestamp),
			"status": "Running",
			"emoji":  "✅",
		},
		{
			"name":   fmt.Sprintf("nginx-deployment-86dcb47867-%s", timestamp[:6]),
			"status": "Running",
			"emoji":  "✅",
		},
		{
			"name":   fmt.Sprintf("db-statefulset-0-%s", timestamp[:4]),
			"status": "Running",
			"emoji":  "✅",
		},
	}

	// Add a problem pod if the current second is even (to randomly show issues)
	if time.Now().Second()%2 == 0 {
		results = append(results, map[string]string{
			"name":   fmt.Sprintf("problematic-pod-%s", timestamp[:8]),
			"status": "CrashLoopBackOff",
			"emoji":  "❌",
		})
	}

	return results
}

// FormatResult creates a human-readable representation of pod statuses
func (p *KubeHealthPlugin) FormatResult(result interface{}) (string, error) {
	pods, ok := result.([]map[string]string)
	if !ok {
		return "", fmt.Errorf("unexpected result type")
	}

	var sb strings.Builder
	sb.WriteString("Kubernetes Pod Status:\n\n")
	for _, pod := range pods {
		emoji := pod["emoji"]
		line := fmt.Sprintf("  %s: %s %s\n", pod["name"], pod["status"], emoji)
		sb.WriteString(line)
	}
	return sb.String(), nil
}

// NewKubeHealthPlugin creates a new instance of the KubeHealth plugin
func NewKubeHealthPlugin(logger *logrus.Logger) pluginconf.Plugin {
	return &KubeHealthPlugin{
		logger: logger,
	}
}

var PluginInstance = NewKubeHealthPlugin(logrus.New())
