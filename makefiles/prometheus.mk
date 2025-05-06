#----------------------------------------
# Prometheus Integration
#----------------------------------------

## Prometheus Monitoring System
.PHONY: prometheus-install prometheus-port-forward prometheus-uninstall \
        prometheus-show-metrics grafana-install grafana-port-forward grafana-uninstall

# Variables for Prometheus installation
PROMETHEUS_NAMESPACE ?= monitoring
PROMETHEUS_RELEASE ?= prometheus
PROMETHEUS_CHART_VERSION ?= 25.8.0
PROMETHEUS_PORT ?= 9090

# Variables for Grafana
GRAFANA_RELEASE ?= grafana
GRAFANA_CHART_VERSION ?= 8.15.0
GRAFANA_PORT ?= 3000

# Install Prometheus using Helm
prometheus-install: ## Install Prometheus in Kubernetes using Helm
	@echo "Installing Prometheus in namespace $(PROMETHEUS_NAMESPACE)..."
	kubectl create namespace $(PROMETHEUS_NAMESPACE) 2>/dev/null || true
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install $(PROMETHEUS_RELEASE) prometheus-community/prometheus \
		--namespace $(PROMETHEUS_NAMESPACE) \
		--version $(PROMETHEUS_CHART_VERSION) \
		--set server.persistentVolume.enabled=false \
		--set alertmanager.persistentVolume.enabled=false
	@echo "Prometheus installed. To access use: make prometheus-port-forward"

# Port forwarding to access Prometheus
prometheus-port-forward: ## Access Prometheus web interface via port-forward
	@echo "Setting up port-forward for Prometheus at http://localhost:$(PROMETHEUS_PORT)"
	kubectl port-forward -n $(PROMETHEUS_NAMESPACE) svc/$(PROMETHEUS_RELEASE)-server $(PROMETHEUS_PORT):80

# Uninstall Prometheus
prometheus-uninstall: ## Uninstall Prometheus from Kubernetes
	@echo "Uninstalling Prometheus..."
	helm uninstall $(PROMETHEUS_RELEASE) -n $(PROMETHEUS_NAMESPACE)

# Show ExpressOps metrics
prometheus-show-metrics: ## Show current ExpressOps metrics
	@echo "Getting ExpressOps metrics..."
	kubectl port-forward svc/expressops 8080:8080 & \
	sleep 2 && curl -s http://localhost:8080/metrics | grep expressops && \
	pkill -f "port-forward svc/expressops"

# Install Grafana
grafana-install: ## Install Grafana with preconfigured dashboard
	@echo "Installing Grafana in namespace $(PROMETHEUS_NAMESPACE)..."
	kubectl create namespace $(PROMETHEUS_NAMESPACE) 2>/dev/null || true
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install $(GRAFANA_RELEASE) grafana/grafana \
		--namespace $(PROMETHEUS_NAMESPACE) \
		--version $(GRAFANA_CHART_VERSION) \
		--set persistence.enabled=false \
		--set admin.password=admin123 \
		--set datasources."datasources\.yaml".apiVersion=1 \
		--set datasources."datasources\.yaml".datasources[0].name=Prometheus \
		--set datasources."datasources\.yaml".datasources[0].type=prometheus \
		--set datasources."datasources\.yaml".datasources[0].url=http://$(PROMETHEUS_RELEASE)-server \
		--set datasources."datasources\.yaml".datasources[0].access=proxy \
		--set datasources."datasources\.yaml".datasources[0].isDefault=true
	@echo "Grafana installed. To access use: make grafana-port-forward"
	@echo "Default credentials: admin / admin123"

# Port forwarding to access Grafana
grafana-port-forward: ## Access Grafana web interface via port-forward
	@echo "Setting up port-forward for Grafana at http://localhost:$(GRAFANA_PORT)"
	kubectl port-forward -n $(PROMETHEUS_NAMESPACE) svc/$(GRAFANA_RELEASE) $(GRAFANA_PORT):80

# Uninstall Grafana
grafana-uninstall: ## Uninstall Grafana from Kubernetes
	@echo "Uninstalling Grafana..."
	helm uninstall $(GRAFANA_RELEASE) -n $(PROMETHEUS_NAMESPACE)