#----------------------------------------
# Prometheus Integration
#----------------------------------------

## Grafana Monitoring System
.PHONY: grafana-install grafana-port-forward grafana-uninstall
# Grafana is connected to Prometheus in Monitoring namespace

# Install Grafana
grafana-install:
	@echo "Installing Grafana in namespace $(PROMETHEUS_NAMESPACE)..."
	kubectl create namespace $(PROMETHEUS_NAMESPACE) 2>/dev/null || true
	helm repo add grafana https://grafana.github.io/helm-charts
	helm repo update
	helm upgrade --install $(GRAFANA_RELEASE) grafana/grafana \
		--namespace $(PROMETHEUS_NAMESPACE) \
		--version $(GRAFANA_CHART_VERSION) \
		--set persistence.enabled=false \
		--set admin.password=expressops \
		--set service.type=ClusterIP \
		--set resources.requests.cpu=10m \
		--set resources.requests.memory=100Mi \
		--set datasources."datasources\.yaml".apiVersion=1 \
		--set datasources."datasources\.yaml".datasources[0].name=Prometheus \
		--set datasources."datasources\.yaml".datasources[0].type=prometheus \
		--set datasources."datasources\.yaml".datasources[0].url=http://prometheus-kube-prometheus-prometheus:9090 \
		--set datasources."datasources\.yaml".datasources[0].access=proxy \
		--set datasources."datasources\.yaml".datasources[0].isDefault=true
	@echo "Grafana installed. To access use: make grafana-port-forward"
	@echo "Credentials: user: admin / passwd: expressops"

# Port forwarding to access Grafana
grafana-port-forward:
	@echo "Setting up port-forward for Grafana at http://localhost:$(GRAFANA_PORT)"
	kubectl port-forward -n $(PROMETHEUS_NAMESPACE) svc/$(GRAFANA_RELEASE) $(GRAFANA_PORT):80

# Uninstall Grafana
grafana-uninstall:
	@echo "Uninstalling Grafana..."
	helm uninstall $(GRAFANA_RELEASE) -n $(PROMETHEUS_NAMESPACE) || true
