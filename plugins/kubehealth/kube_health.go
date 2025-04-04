// plugins/kubehealth/kube_health.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("error loading kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client: %v", err)
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %v", err)
	}

	results := make([]map[string]string, 0)
	for _, pod := range pods.Items {
		status := string(pod.Status.Phase)
		if pod.Status.Phase == v1.PodRunning {
			for _, c := range pod.Status.ContainerStatuses {
				if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
					status = "CrashLoopBackOff"
					break
				}
			}
		}

		results = append(results, map[string]string{
			"name":   pod.Name,
			"status": status,
		})
	}

	return results, nil
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
		line := fmt.Sprintf("  %s: %s\n", pod["name"], pod["status"])
		if pod["status"] == "CrashLoopBackOff" {
			line = fmt.Sprintf("  %s: \033[31m%s\033[0m ❌\n", pod["name"], pod["status"])
		}
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
