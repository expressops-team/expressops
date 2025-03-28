// plugins/kubehealth/kube_health.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type KubeHealthPlugin struct {
	logger         *logrus.Logger
	k8sClient      K8sClient
	targetPods     []string
	targetServices []string
}

type K8sClient interface {
	GetPodStatus(ctx context.Context, namespace, podName string) (string, error)
	GetServiceStatus(ctx context.Context, namespace, serviceName string) (string, error)
}

type MockK8sClient struct{}

func (m *MockK8sClient) GetPodStatus(ctx context.Context, namespace, podName string) (string, error) {
	// Simular pod status
	statuses := map[string]string{
		"frontend": "Running",
		"backend":  "Running",
		"database": "Running",
		"cache":    "Running",
		"failing":  "CrashLoopBackOff",
	}

	// Comprobar si el contexto ha sido cancelado
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Simular un pequeño retraso
		time.Sleep(100 * time.Millisecond)
		if status, ok := statuses[podName]; ok {
			return status, nil
		}
		return "Unknown", nil
	}
}

func (m *MockK8sClient) GetServiceStatus(ctx context.Context, namespace, serviceName string) (string, error) {
	// Simular service status
	statuses := map[string]string{
		"api-gateway":  "Healthy",
		"auth":         "Healthy",
		"payments":     "Degraded",
		"notification": "Healthy",
	}

	// Comprobar si el contexto ha sido cancelado
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Simular un pequeño retraso
		time.Sleep(100 * time.Millisecond)
		if status, ok := statuses[serviceName]; ok {
			return status, nil
		}
		return "Unknown", nil
	}
}

func NewKubeHealthPlugin(logger *logrus.Logger) *KubeHealthPlugin {
	return &KubeHealthPlugin{
		logger:    logger,
		k8sClient: &MockK8sClient{},
		targetPods: []string{
			"frontend",
			"backend",
			"database",
			"cache",
			"failing",
		},
		targetServices: []string{
			"api-gateway",
			"auth",
			"payments",
			"notification",
		},
	}
}

func (p *KubeHealthPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger

	// Configure targets from config if provided
	if pods, ok := config["target_pods"].([]interface{}); ok && len(pods) > 0 {
		p.targetPods = make([]string, 0, len(pods))
		for _, pod := range pods {
			if podName, ok := pod.(string); ok {
				p.targetPods = append(p.targetPods, podName)
			}
		}
	}

	if services, ok := config["target_services"].([]interface{}); ok && len(services) > 0 {
		p.targetServices = make([]string, 0, len(services))
		for _, svc := range services {
			if svcName, ok := svc.(string); ok {
				p.targetServices = append(p.targetServices, svcName)
			}
		}
	}

	p.logger.Info("Initializing Kubernetes Health Plugin")
	return nil
}

func (p *KubeHealthPlugin) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	p.logger.Info("Executing Kubernetes Health Check")

	if r != nil {
		p.logger.Infof("Kubernetes health check request from: %s", r.RemoteAddr)

		// Check query parameters
		if namespace := r.URL.Query().Get("namespace"); namespace != "" {
			p.logger.Infof("Using namespace from request: %s", namespace)
		}

		// Allow overriding targets via query parameters
		if podsParam := r.URL.Query().Get("pods"); podsParam != "" {
			p.targetPods = strings.Split(podsParam, ",")
			p.logger.Infof("Using pods from request: %v", p.targetPods)
		}

		if servicesParam := r.URL.Query().Get("services"); servicesParam != "" {
			p.targetServices = strings.Split(servicesParam, ",")
			p.logger.Infof("Using services from request: %v", p.targetServices)
		}
	}

	// Also check shared context for parameters
	if shared != nil {
		if namespace, ok := (*shared)["namespace"].(string); ok && namespace != "" {
			p.logger.Infof("Using namespace from shared context: %s", namespace)
		}

		if pods, ok := (*shared)["pods"].(string); ok && pods != "" {
			p.targetPods = strings.Split(pods, ",")
			p.logger.Infof("Using pods from shared context: %v", p.targetPods)
		} else if podsArr, ok := (*shared)["pods"].([]interface{}); ok && len(podsArr) > 0 {
			p.targetPods = make([]string, 0, len(podsArr))
			for _, pod := range podsArr {
				if podStr, ok := pod.(string); ok {
					p.targetPods = append(p.targetPods, podStr)
				}
			}
			p.logger.Infof("Using pods from shared context: %v", p.targetPods)
		}

		if services, ok := (*shared)["services"].(string); ok && services != "" {
			p.targetServices = strings.Split(services, ",")
			p.logger.Infof("Using services from shared context: %v", p.targetServices)
		} else if servicesArr, ok := (*shared)["services"].([]interface{}); ok && len(servicesArr) > 0 {
			p.targetServices = make([]string, 0, len(servicesArr))
			for _, svc := range servicesArr {
				if svcStr, ok := svc.(string); ok {
					p.targetServices = append(p.targetServices, svcStr)
				}
			}
			p.logger.Infof("Using services from shared context: %v", p.targetServices)
		}
	}

	namespace := "default" // Default namespace

	// Process pods
	podResults := make(map[string]string)
	for _, pod := range p.targetPods {
		status, err := p.k8sClient.GetPodStatus(ctx, namespace, pod)
		if err != nil {
			p.logger.Warnf("Error getting status for pod %s: %v", pod, err)
			podResults[pod] = "Error"
		} else {
			podResults[pod] = status
		}
	}

	// Process services
	serviceResults := make(map[string]string)
	for _, service := range p.targetServices {
		status, err := p.k8sClient.GetServiceStatus(ctx, namespace, service)
		if err != nil {
			p.logger.Warnf("Error getting status for service %s: %v", service, err)
			serviceResults[service] = "Error"
		} else {
			serviceResults[service] = status
		}
	}

	// Calculate overall health
	overallHealth := "Healthy"
	for _, status := range podResults {
		if status != "Running" {
			overallHealth = "Degraded"
			break
		}
	}

	if overallHealth == "Healthy" {
		for _, status := range serviceResults {
			if status != "Healthy" {
				overallHealth = "Degraded"
				break
			}
		}
	}

	// Count unhealthy components
	unhealthyPods := 0
	for _, status := range podResults {
		if status != "Running" {
			unhealthyPods++
		}
	}

	unhealthyServices := 0
	for _, status := range serviceResults {
		if status != "Healthy" {
			unhealthyServices++
		}
	}

	// Return structured results
	return map[string]interface{}{
		"status":    overallHealth,
		"pods":      podResults,
		"services":  serviceResults,
		"namespace": namespace,
		"unhealthy_count": map[string]interface{}{
			"pods":     unhealthyPods,
			"services": unhealthyServices,
		},
	}, nil
}

func (p *KubeHealthPlugin) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		var formattedOutput strings.Builder

		// Add overall status with emoji
		overallStatus, _ := resultMap["status"].(string)
		var statusEmoji string
		switch overallStatus {
		case "Healthy":
			statusEmoji = "💚"
		case "Degraded":
			statusEmoji = "💛"
		default:
			statusEmoji = "❤️"
		}

		formattedOutput.WriteString(fmt.Sprintf("%s Kubernetes Cluster Status: %s\n\n", statusEmoji, overallStatus))

		// Add pod statuses
		formattedOutput.WriteString("Kubernetes Pod Status:\n")
		if pods, ok := resultMap["pods"].(map[string]string); ok {
			for pod, status := range pods {
				var podEmoji string
				switch status {
				case "Running":
					podEmoji = "✅"
				case "CrashLoopBackOff":
					podEmoji = "❌"
				default:
					podEmoji = "⚠️"
				}
				formattedOutput.WriteString(fmt.Sprintf("  %s %s: %s\n", podEmoji, pod, status))
			}
		}

		// Add service statuses
		formattedOutput.WriteString("\nKubernetes Service Status:\n")
		if services, ok := resultMap["services"].(map[string]string); ok {
			for service, status := range services {
				var serviceEmoji string
				switch status {
				case "Healthy":
					serviceEmoji = "✅"
				case "Degraded":
					serviceEmoji = "⚠️"
				default:
					serviceEmoji = "❌"
				}
				formattedOutput.WriteString(fmt.Sprintf("  %s %s: %s\n", serviceEmoji, service, status))
			}
		}

		// Add summary
		if unhealthyCount, ok := resultMap["unhealthy_count"].(map[string]interface{}); ok {
			unhealthyPods, _ := unhealthyCount["pods"].(int)
			unhealthyServices, _ := unhealthyCount["services"].(int)

			if unhealthyPods > 0 || unhealthyServices > 0 {
				formattedOutput.WriteString("\n⚠️ Summary of issues:\n")
				if unhealthyPods > 0 {
					formattedOutput.WriteString(fmt.Sprintf("  - %d pods are not in Running state\n", unhealthyPods))
				}
				if unhealthyServices > 0 {
					formattedOutput.WriteString(fmt.Sprintf("  - %d services are not Healthy\n", unhealthyServices))
				}
			} else {
				formattedOutput.WriteString("\n✨ All components are healthy!\n")
			}
		}

		return formattedOutput.String(), nil
	}

	return "Kubernetes health check completed", nil
}

var PluginInstance pluginconf.Plugin = NewKubeHealthPlugin(nil)
