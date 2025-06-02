# ExpressOps Makefile
# Main entry point for all operations
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
BOLD = \033[1m
RESET = \033[0m
PRINT = @echo 

# Common variables
IMAGE_REPOSITORY ?= davidnull/expressops
IMAGE_TAG ?= 1.1.8
PLUGINS_PATH ?= plugins
CONTAINER_NAME ?= expressops-app
HOST_PORT ?= 8080
SERVER_PORT ?= 8080
SERVER_ADDRESS ?= 0.0.0.0
TIMEOUT_SECONDS ?= 4
LOG_LEVEL ?= info	
LOG_FORMAT ?= text
SLACK_WEBHOOK_URL ?= external-secret-webhook
CONFIG_PATH ?= docs/samples/config.yaml
CONFIG_MOUNT_PATH ?= /app/config.yaml
K8S_NAMESPACE ?= default
GCP_SA_KEY_FILE ?= key.json
KUBECONFIG ?= ~/.kube/config

# Prometheus/Grafana variables
PROMETHEUS_NAMESPACE ?= monitoring # Namespace for Grafana. Assumes your existing Prometheus (prometheus-kube-prometheus-prometheus) is also in this namespace.
GRAFANA_RELEASE ?= david-grafana
GRAFANA_CHART_VERSION ?= 8.15.0
GRAFANA_PORT ?= 3000

# Include other makefiles
include makefiles/docker.mk
include makefiles/kubernetes.mk
include makefiles/helm.mk
include makefiles/build.mk
include makefiles/prometheus.mk


# help
# @{} is for output like more command (less -R)
help:
	@{ \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(YELLOW)$(BOLD)                     ExpressOps - Kubernetes Deployment System$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BLUE)Press $(RED)'q'$(BLUE) to exit this view$(RESET)"; \
		echo; \
		echo "$(BLUE)$(BOLD)MOST FREQUENTLY USED COMMANDS:$(RESET)"; \
		echo "  $(RED)make setup-with-gcp-credentials$(RESET)  - Complete setup with GCP credentials"; \
		echo "  $(GREEN)make helm-install-with-gcp-secrets$(RESET) - Deploy using GCP secrets"; \
		echo "  $(GREEN)make k8s-port-forward$(RESET)           - Access the application"; \
		echo "  $(GREEN)make k8s-status$(RESET)                 - Check deployment status"; \
		echo "  $(GREEN)make k8s-logs$(RESET)                   - View application logs"; \
		echo; \
		echo "$(BLUE)$(BOLD)MONITORING COMMANDS (Grafana connects to existing Prometheus):$(RESET)"; \
		echo "  $(GREEN)make grafana-install$(RESET)            - Install Grafana (will connect to prometheus-kube-prometheus-prometheus in PROMETHEUS_NAMESPACE)"; \
		echo "  $(GREEN)make grafana-port-forward$(RESET)       - Access Grafana UI (http://localhost:$(GRAFANA_PORT))"; \
		echo; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo; \
		for mkfile in $(sort $(MAKEFILE_LIST)); do \
			if [ "$$mkfile" = "Makefile" ]; then \
				echo "$(BOLD)$(BLUE)Main Commands:$(RESET)"; \
			elif [ "$$mkfile" = "makefiles/build.mk" ]; then \
				echo "$(BOLD)$(BLUE)Development Commands:$(RESET)"; \
			elif [ "$$mkfile" = "makefiles/docker.mk" ]; then \
				echo "$(BOLD)$(BLUE)Docker Commands:$(RESET)"; \
			elif [ "$$mkfile" = "makefiles/kubernetes.mk" ]; then \
				echo "$(BOLD)$(BLUE)Kubernetes Commands:$(RESET)"; \
			elif [ "$$mkfile" = "makefiles/helm.mk" ]; then \
				echo "$(BOLD)$(BLUE)Helm Commands:$(RESET)"; \
			elif [ "$$mkfile" = "makefiles/prometheus.mk" ]; then \
				echo "$(BOLD)$(BLUE)Monitoring Commands:$(RESET)"; \
			else \
				echo "$(BOLD)$(BLUE)$$mkfile:$(RESET)"; \
			fi; \
			grep -E '^## .*$$' $$mkfile | awk 'BEGIN {FS = "## "}; {printf "  $(YELLOW)%s$(RESET)\n", $$2}'; \
			echo; \
			grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $$mkfile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    $(GREEN)%-28s$(RESET) %s\n", $$1, $$2}'; \
			echo; \
		done; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BOLD)$(BLUE)Development Workflow:$(RESET)"; \
		echo "  1. $(GREEN)make build$(RESET) - Build the application locally"; \
		echo "  2. $(GREEN)make run$(RESET) - Run the application locally"; \
		echo "  3. $(GREEN)make docker-build$(RESET) - Create Docker image"; \
		echo "  4. $(GREEN)make k8s-install-eso$(RESET) - Install External Secrets Operator"; \
		echo "  5. $(GREEN)make helm-install-with-gcp-secrets$(RESET) - Deploy to Kubernetes"; \
		echo; \
		echo "$(BOLD)$(BLUE)Monitoring Workflow (Grafana with existing Prometheus):$(RESET)"; \
		echo "  1. $(GREEN)make grafana-install$(RESET) (Ensure PROMETHEUS_NAMESPACE is set to where your existing Prometheus and Grafana will run)"; \
		echo "  2. $(GREEN)make grafana-port-forward$(RESET)"; \
		echo "  3. $(GREEN)Access Grafana at http://localhost:$(GRAFANA_PORT) with admin/expressops$(RESET)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Google Cloud Secret Manager:$(RESET)"; \
		echo "  Account: $(GREEN)expressops-external-secrets@fc-it-school-2025.iam.gserviceaccount.com$(RESET)"; \
		echo "  Secret: $(GREEN)projects/88527591198/secrets/slack-webhook$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
	} | less -R 

## With less you can go back and forth with the help menu
## Shows configuration values
config:
	@{ \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(YELLOW)$(BOLD)                     ExpressOps - Configuration Values$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BLUE)Press $(RED)'q'$(BLUE) to exit this view$(RESET)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Runtime Configuration:$(RESET)"; \
		echo "  $(GREEN)SERVER_PORT$(RESET)       = $(SERVER_PORT)"; \
		echo "  $(GREEN)SERVER_ADDRESS$(RESET)    = $(SERVER_ADDRESS)"; \
		echo "  $(GREEN)TIMEOUT_SECONDS$(RESET)   = $(TIMEOUT_SECONDS)"; \
		echo "  $(GREEN)LOG_LEVEL$(RESET)         = $(LOG_LEVEL)"; \
		echo "  $(GREEN)LOG_FORMAT$(RESET)        = $(LOG_FORMAT)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Docker Configuration:$(RESET)"; \
		echo "  $(GREEN)IMAGE_REPOSITORY$(RESET)  = $(IMAGE_REPOSITORY)"; \
		echo "  $(GREEN)IMAGE_TAG$(RESET)         = $(IMAGE_TAG)"; \
		echo "  $(GREEN)CONTAINER_NAME$(RESET)    = $(CONTAINER_NAME)"; \
		echo "  $(GREEN)HOST_PORT$(RESET)         = $(HOST_PORT)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Kubernetes Configuration:$(RESET)"; \
		echo "  $(GREEN)K8S_NAMESPACE$(RESET)     = $(K8S_NAMESPACE)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Monitoring Configuration:$(RESET)"; \
		echo "  $(GREEN)PROMETHEUS_NAMESPACE$(RESET)  = $(PROMETHEUS_NAMESPACE)"; \
		echo "  $(GREEN)GRAFANA_RELEASE$(RESET)       = $(GRAFANA_RELEASE)"; \
		echo "  $(GREEN)GRAFANA_CHART_VERSION$(RESET)  = $(GRAFANA_CHART_VERSION)"; \
		echo "  $(GREEN)GRAFANA_PORT$(RESET)          = $(GRAFANA_PORT)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Secrets Configuration:$(RESET)"; \
		echo "  $(GREEN)SLACK_WEBHOOK_URL$(RESET) = $(SLACK_WEBHOOK_URL)"; \
		echo "  $(GREEN)GCP_SA_KEY_FILE$(RESET)   = $(GCP_SA_KEY_FILE)"; \
		echo; \
		echo "$(BOLD)$(BLUE)File Paths:$(RESET)"; \
		echo "  $(GREEN)PLUGINS_PATH$(RESET)      = $(PLUGINS_PATH)"; \
		echo "  $(GREEN)CONFIG_PATH$(RESET)       = $(CONFIG_PATH)"; \
		echo "  $(GREEN)CONFIG_MOUNT_PATH$(RESET) = $(CONFIG_MOUNT_PATH)"; \
		echo; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BOLD)To change these values, set the corresponding environment variable or edit the Makefile.$(RESET)"; \
		echo "Example: $(GREEN)IMAGE_TAG=2.0.0 make docker-build$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
	} | less -R

## Shows information about the project
about:
	@{ \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(YELLOW)$(BOLD)                           ExpressOps$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BLUE)Press $(RED)'q'$(BLUE) to exit this view$(RESET)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Project Overview:$(RESET)"; \
		echo "  ExpressOps is the Tech School's project for the year 2025"; \
		echo "  It is a Kubernetes deployment system for managing operations"; \
		echo "  with external secrets from Google Cloud Secret Manager."; \
		echo; \
		echo "$(BOLD)$(BLUE)Getting Started:$(RESET)"; \
		echo "  1. Build locally: $(GREEN)make build$(RESET)"; \
		echo "  2. Run locally: $(GREEN)make run$(RESET)"; \
		echo "  3. Deploy to K8s: $(GREEN)make setup-with-gcp-credentials$(RESET)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Monitoring:$(RESET)"; \
		echo "  1. Install Grafana: $(GREEN)make grafana-install$(RESET)"; \
		echo "  2. Access Grafana: $(GREEN)make grafana-port-forward$(RESET)"; \
		echo; \
		echo "$(BOLD)$(BLUE)Documentation:$(RESET)"; \
		echo "  • For help: $(GREEN)make help$(RESET)"; \
		echo "  • Quick reference: $(GREEN)make quick-help$(RESET)"; \
		echo "  • Configuration: $(GREEN)make config$(RESET)"; \
		echo; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BOLD)Need help? Start with: $(GREEN)make quick-help$(RESET)$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
	} | less -R

## Quick help with most common commands
quick-help:
	@{ \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BOLD)$(BLUE)ExpressOps Quick Help$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "$(BLUE)Press $(RED)'q'$(BLUE) to exit this view$(RESET)"; \
		echo; \
		echo "$(BOLD)Development:$(RESET)"; \
		echo "  $(GREEN)make build$(RESET)                    - Build the application"; \
		echo "  $(GREEN)make run$(RESET)                      - Run locally"; \
		echo; \
		echo "$(BOLD)Docker:$(RESET)"; \
		echo "  $(GREEN)make docker-build$(RESET)             - Build container"; \
		echo "  $(GREEN)make docker-run$(RESET)               - Run container locally"; \
		echo; \
		echo "$(BOLD)Kubernetes:$(RESET)"; \
		echo "  $(RED)make setup-with-gcp-credentials$(RESET) - Complete setup with GCP"; \
		echo "  $(GREEN)make helm-install-with-gcp-secrets$(RESET) - Deploy with GCP secrets"; \
		echo "  $(GREEN)make k8s-status$(RESET)               - Check deployment status"; \
		echo "  $(GREEN)make k8s-logs$(RESET)                 - View application logs"; \
		echo "  $(GREEN)make k8s-port-forward$(RESET)         - Access the application"; \
		echo; \
		echo "$(BOLD)Monitoring (Grafana with existing Prometheus):$(RESET)"; \
		echo "  $(GREEN)make grafana-install$(RESET)                  - Install Grafana (connects to prometheus-kube-prometheus-prometheus)"; \
		echo "  $(GREEN)make grafana-port-forward$(RESET)             - Access Grafana UI (http://localhost:$(GRAFANA_PORT))"; \
		echo; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
		echo "For full help: $(GREEN)make help$(RESET)"; \
		echo "$(YELLOW)=================================================================================$(RESET)"; \
	} | less -R

# Easy installation with custom kubectl config
setup-with-custom-kubectl: ## Setup with custom kubectl configuration
	@echo "$(BLUE)🔄 Setting up ExpressOps with custom kubectl configuration...$(RESET)"
	@echo "$(YELLOW)Using KUBECONFIG: $(KUBECONFIG)$(RESET)"
	@KUBECONFIG=$(KUBECONFIG) make setup-with-gcp-credentials
	@echo "$(GREEN)✅ Setup complete with custom kubectl configuration$(RESET)"

.DEFAULT_GOAL := help

# Ensure makefiles exists
$(shell mkdir -p makefiles) 
