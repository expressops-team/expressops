# ---------------------------------------------
#  ExpressOps — Makefile                       #
# ---------------------------------------------
.DEFAULT_GOAL := help

# ---------------------------------------------
# 🎨 Colors
GREEN  = \033[32m
RED    = \033[31m
BLUE   = \033[34m
YELLOW = \033[33m
BOLD   = \033[1m
RESET  = \033[0m
PRINT  = @echo

# ---------------------------------------------
# 🔧 Configurable variables
IMAGE_REPOSITORY   ?= expressopsfreepik/expressops
IMAGE_NAME         ?= expressops
IMAGE_TAG          ?= 1.1.8
CONTAINER_NAME     ?= expressops-app
HOST_PORT          ?= 8080
SERVER_PORT        ?= 8080
SERVER_ADDRESS     ?= 0.0.0.0
TIMEOUT_SECONDS    ?= 4
LOG_LEVEL          ?= info
LOG_FORMAT         ?= text
SLACK_WEBHOOK_URL  ?= external-secret-webhook
PLUGINS_PATH       ?= plugins
CONFIG_PATH        ?= docs/samples/config.yaml
CONFIG_MOUNT_PATH  ?= /app/config.yaml
HELM_CHART_DIR     ?= k3s/expressops-chart
HELM_VALUES_FILE   ?= $(HELM_CHART_DIR)/values.yaml
DOCKER_TAG_FILE    = .docker_tag

# Kubernetes variables
K8S_NAMESPACE      ?= expressops-dev
KUBECONFIG         ?= ~/.kube/config
GCP_SA_KEY_FILE    ?= key.json

# Prometheus/Grafana variables

PROMETHEUS_NAMESPACE ?= monitoring # Namespace for Grafana. Assumes your existing Prometheus (prometheus-kube-prometheus-prometheus) is also in this namespace.
GRAFANA_RELEASE ?= grafana
GRAFANA_CHART_VERSION ?= 8.15.0
GRAFANA_PORT         ?= 3000

# ---------------------------------------------
# Import modular makefiles
-include makefiles/*.mk

# ---------------------------------------------
# 📜 Goals (.PHONY)
.PHONY: help

# ---------------------------------------------
# 📦 Release end-to-end
release: docker-build docker-push helm-deploy
	@echo "✅ Release process completed."

# Función para generar auto-tag
generate-new-tag:
	@echo "🐳 Checking existing Docker image versions..."
	@VERSION=$$(docker images --format "{{.Tag}}" $(IMAGE_NAME) | grep -E '^v[0-9]+$$' | sed 's/v//' | sort -n | tail -n1); \
	if [ -z "$$VERSION" ]; then \
		NEXT_VERSION=1; \
	else \
		NEXT_VERSION=$$((VERSION + 1)); \
	fi; \
	NEW_TAG=v$$NEXT_VERSION; \
	echo "$$NEW_TAG" > .docker_tag; \
	echo "📦 Generated new tag: $$NEW_TAG";

# Auto-versioning de imagen y actualización de Helm values
update-helm-values:
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || echo "latest"); \
	echo "🔄 Updating Helm values file: $(HELM_VALUES_FILE) with tag: $$NEW_TAG"; \
	yq e '.image.tag = "'$$NEW_TAG'"' -i $(HELM_VALUES_FILE); \
	echo "✅ Helm values file updated."

# Run Docker container with SRE2 configuration
docker-run-sre2:
	$(MAKE) docker-run CONFIG_PATH=docs/samples/config_SRE2.yaml CONFIG_MOUNT_PATH=/app/config.yaml

# ---------------------------------------------
# 📖 Help
help:
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo "$(YELLOW)$(BOLD)                     ExpressOps - Kubernetes Deployment System$(RESET)"
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo ""
	@echo "Available commands:"
	@echo "================================================"
	@echo "  $(BLUE)Basic Commands:$(RESET)"
	@echo "    make help                   - Show this help"
	@echo "    make build                  - Build plugins and application locally"
	@echo "    make run                    - Run application locally"
	@echo ""
	@echo "  $(BLUE)Docker Workflow:$(RESET)"
	@echo "    make docker-build           - Build Docker image (auto-versioned) and updates Helm values.yaml"
	@echo "    make docker-run             - Run container with the last built tag"
	@echo "    make docker-run-build       - Build and run Docker container"
	@echo "    make docker-clean           - Clean Docker resources (stops/removes container, old tags)"
	@echo "    make docker-push            - Tag and push the last built image to Docker Hub"
	@echo "    make docker-run-sre2        - Run container with SRE2 configuration"
	@echo ""
	@echo "  $(BLUE)Helm/Kubernetes Workflow:$(RESET)"
	@echo "    make helm-deploy            - Deploy/Upgrade Helm chart using tag from values.yaml"
	@echo "    make helm-install-with-gcp-secrets - Deploy using GCP secrets"
	@echo "    make k8s-port-forward       - Access the application"
	@echo "    make k8s-status             - Check deployment status"
	@echo "    make k8s-logs               - View application logs"
	@echo ""
	@echo "  $(BLUE)Monitoring Commands:$(RESET)"
	@echo "    make grafana-install        - Install Grafana"
	@echo "    make grafana-port-forward   - Access Grafana UI (http://localhost:$(GRAFANA_PORT))"
	@echo ""
	@echo "  $(BLUE)Combined Release Workflow:$(RESET)"
	@echo "    make release                - Full cycle: docker-build, docker-push, helm-deploy"
	@echo "================================================"
	@echo ""
	@echo "Configurable variables (current values):"
	@echo "  IMAGE_REPOSITORY           = $(IMAGE_REPOSITORY)"
	@echo "  IMAGE_NAME                 = $(IMAGE_NAME)"
	@echo "  CONTAINER_NAME             = $(CONTAINER_NAME)"
	@echo "  HOST_PORT                  = $(HOST_PORT)"
	@echo "  SERVER_PORT                = $(SERVER_PORT)"
	@echo "  SERVER_ADDRESS             = $(SERVER_ADDRESS)"
	@echo "  TIMEOUT_SECONDS            = $(TIMEOUT_SECONDS)"
	@echo "  LOG_LEVEL                  = $(LOG_LEVEL)"
	@echo "  LOG_FORMAT                 = $(LOG_FORMAT)"
	@echo "  CONFIG_PATH                = $(CONFIG_PATH)"
	@echo "  CONFIG_MOUNT_PATH          = $(CONFIG_MOUNT_PATH)"
	@echo "  HELM_CHART_DIR             = $(HELM_CHART_DIR)"
	@echo "  HELM_VALUES_FILE           = $(HELM_VALUES_FILE)"
	@echo "  K8S_NAMESPACE              = $(K8S_NAMESPACE)"
	@echo "  PROMETHEUS_NAMESPACE       = $(PROMETHEUS_NAMESPACE)"
	@echo "  GRAFANA_PORT               = $(GRAFANA_PORT)"
