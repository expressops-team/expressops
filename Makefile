# docker run -d --name expressops-demo -p 8080:8080 expressops:Vx   <-- for testing
#      make docker-build   //    make docker-run
# Common commands:
#   make build          - Build plugins and app
#   make run            - Run locally
#   make docker-build   - Build Docker image (auto-versioned)
#   make docker-run     - Run Docker container
#   make docker-clean   - Cleanup Docker
#   make help           - Show command help

# make run <===
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
RESET = \033[0m
PRINT = @echo 

# Configurable variables (can be overridden with environment variables)
IMAGE_NAME ?= expressops
NEW_TAG :=
CONTAINER_NAME ?= expressops-app
HOST_PORT ?= 8080
SERVER_PORT ?= 8080
SERVER_ADDRESS ?= 0.0.0.0
TIMEOUT_SECONDS ?= 4
LOG_LEVEL ?= info
LOG_FORMAT ?= text
SLACK_WEBHOOK_URL ?= 
CONFIG_PATH ?= docs/samples/config.yaml
CONFIG_MOUNT_PATH ?= /app/config.yaml
K8S_NAMESPACE ?= default

.PHONY: build run docker-build docker-push docker-run docker-clean help k8s-deploy k8s-status k8s-logs k8s-delete k8s-port-forward k8s-generate-secrets

# Build plugins and application locally
build:
	@echo "Cleaning previous plugins..."
	@find plugins -name "*.so" -delete
	@echo "Building plugins..."
	@for dir in $$(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
		for gofile in $$dir/*.go; do \
			if [ -f "$$gofile" ]; then \
				plugin_name=$$(basename "$$gofile" .go); \
				echo "Building plugin $$plugin_name.so from $$gofile"; \
				CGO_ENABLED=1 GOOS=linux go build -buildmode=plugin -o "$$dir/$$plugin_name.so" "$$gofile" || exit 1; \
			fi \
		done \
	done
	@echo "Verifying compiled plugins:"
	@find plugins -name "*.so" | sort
	@echo "Building main application..."
	@go build -o expressops ./cmd
	@echo "✅ Build completed"

# Run the application locally
run: build
	@echo "🚀 Starting ExpressOps"
	./expressops -config $(CONFIG_PATH)

# Auto-versioned images Docker build
docker-build:
	@echo "🐳 Checking existing Docker image versions..."
	@VERSION=$$(docker images --format "{{.Tag}}" $(IMAGE_NAME) | grep -E '^v[0-9]+$$' | sed 's/v//' | sort -n | tail -n1); \
	if [ -z "$$VERSION" ]; then \
		NEXT_VERSION=1; \
	else \
		NEXT_VERSION=$$((VERSION + 1)); \
	fi; \
	NEW_TAG=v$$NEXT_VERSION; \
	echo "$$NEW_TAG" > .docker_tag; \
	echo "📦 Building Docker image with tag: $$NEW_TAG"; \
	docker build \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg SERVER_ADDRESS=$(SERVER_ADDRESS) \
		--build-arg TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		--build-arg LOG_LEVEL=$(LOG_LEVEL) \
		--build-arg LOG_FORMAT=$(LOG_FORMAT) \
		--build-arg CONFIG_PATH=$(CONFIG_MOUNT_PATH) \
		-t $(IMAGE_NAME):latest .
	@echo "✅ Image built: $(IMAGE_NAME):latest"

# Run Docker container
docker-run:
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || echo "latest"); \
	echo "🚀 Starting container with tag: $$NEW_TAG"; \
	@echo "📌 Application available at http://localhost:$(HOST_PORT)"; \
	docker run --name $(CONTAINER_NAME) \
		-p $(HOST_PORT):$(SERVER_PORT) \
		-e SERVER_PORT=$(SERVER_PORT) \
		-e SERVER_ADDRESS=$(SERVER_ADDRESS) \
		-e TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		-e LOG_LEVEL=$(LOG_LEVEL) \
		-e LOG_FORMAT=$(LOG_FORMAT) \
		-e SLACK_WEBHOOK_URL=$(SLACK_WEBHOOK_URL) \
		-v $(PWD)/$(CONFIG_PATH):$(CONFIG_MOUNT_PATH) \
		--rm $(IMAGE_NAME):$$NEW_TAG
	@echo "🐳 Running image version: $$NEW_TAG"



# Run Docker container with build
docker-run-build: docker-build docker-run
# Clean Docker resources
docker-clean:
	@echo "🧹 Cleaning Docker resources..."
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	-rm -f .docker_tag
	@echo "🗑 Removing unused images..."
	@echo "🗑 Removing <none> images..."
	-docker rmi $$(docker images -f "dangling=true" -q) 2>/dev/null || true
	@echo "✅ Cleanup completed"

# Run Docker container with SRE2 configuration
docker-run-sre2:
	$(MAKE) docker-run CONFIG_PATH=docs/samples/config_SRE2.yaml CONFIG_MOUNT_PATH=/app/config.yaml


# Help
help:
	@echo "Available commands:"
	@echo "================================================"
	@echo "  make help          - Show this help"
	@echo "  make build         - Build plugins and application"
	@echo "  make run           - Run application locally"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-push   - Build, tag and push Docker image to Docker Hub"
	@echo "  make docker-run    - Run container"

	@echo "  make k8s-deploy    - Deploy to Kubernetes"
	@echo "  make k8s-status    - Check Kubernetes deployment status"
	@echo "  make k8s-logs      - View Kubernetes logs"
	@echo "  make k8s-port-forward - Port forward to access the application"
	@echo "  make k8s-delete    - Delete Kubernetes deployment"
	@echo "  make k8s-generate-secrets - Generate secrets file from template"
	@echo
	@echo "================================================"

	@echo "Configurable variables (current values):"
	@echo "  IMAGE_NAME       = $(IMAGE_NAME)"
	@echo "  CONTAINER_NAME   = $(CONTAINER_NAME)"
	@echo "  HOST_PORT        = $(HOST_PORT)"
	@echo "  SERVER_PORT      = $(SERVER_PORT)"
	@echo "  SERVER_ADDRESS   = $(SERVER_ADDRESS)"
	@echo "  TIMEOUT_SECONDS  = $(TIMEOUT_SECONDS)"
	@echo "  LOG_LEVEL        = $(LOG_LEVEL)"
	@echo "  LOG_FORMAT       = $(LOG_FORMAT)"
	@echo "  SLACK_WEBHOOK_URL = $(SLACK_WEBHOOK_URL)"
	@echo "  CONFIG_PATH      = $(CONFIG_PATH)"
	@echo "  CONFIG_MOUNT_PATH = $(CONFIG_MOUNT_PATH)"
	@echo "  K8S_NAMESPACE    = $(K8S_NAMESPACE)"

# Generate secrets file from template
k8s-generate-secrets:
	@if [ ! -f k8s/secrets.yaml ]; then \
		echo "⚠️ secrets.yaml does not exist, creating from example..."; \
		cp k8s/secrets.example.yaml k8s/secrets.yaml; \
		echo "✅ k8s/secrets.yaml created. Edit it with the actual values."; \
	else \
		echo "✅ k8s/secrets.yaml already exists."; \
	fi
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "⚠️ SLACK_WEBHOOK_URL is not set in environment variables."; \
		echo "   Edit k8s/secrets.yaml manually or configure SLACK_WEBHOOK_URL."; \
	else \
		echo "✅ Updating SLACK_WEBHOOK_URL in k8s/secrets.yaml..."; \
		sed -i "s|https://hooks.slack.com/services/.*|$(SLACK_WEBHOOK_URL)\"|" k8s/secrets.yaml; \
	fi

# Kubernetes Deployment
# Before deploying, make sure to:
# 1. Set the SLACK_WEBHOOK_URL in the Makefile or environment (optional)
# 2. Connect to Kubernetes with the SSH tunnel:
#    gcloud compute ssh --zone "europe-west1-d" "it-school-2025-1" --tunnel-through-iap --project "fc-it-school-2025" --ssh-flag "-N -L 6443:127.0.0.1:6443"
# 3. Build and push the image to Docker Hub (optional):
#    make docker-push
k8s-deploy: k8s-generate-secrets
	@echo "🔄 Deploying ExpressOps to Kubernetes..."
	@echo "📦 Applying Kubernetes resources..."
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secrets.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	@echo "✅ ExpressOps deployed to Kubernetes"
	@echo "🔍 Check status with: make k8s-status"
	@echo "🌐 Access the application with: make k8s-port-forward"

k8s-status:
	@echo "🔍 Checking ExpressOps deployment status:"
	@kubectl get pods -l app=expressops -n $(K8S_NAMESPACE)
	@echo "\n🌐 Service status:"
	@kubectl get svc expressops -n $(K8S_NAMESPACE)
	@echo "\n📊 Deployment status:"
	@kubectl get deployment expressops -n $(K8S_NAMESPACE)

k8s-logs:
	@echo "📃 ExpressOps logs:"
	@POD=$$(kubectl get pods -l app=expressops -n $(K8S_NAMESPACE) -o jsonpath="{.items[0].metadata.name}"); \
	if [ -n "$$POD" ]; then \
		kubectl logs $$POD -n $(K8S_NAMESPACE) --tail=100; \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-port-forward:
	@echo "🔄 Setting up port forwarding for ExpressOps service..."
	@echo "🌐 Access the application at http://localhost:$(HOST_PORT)"
	@POD=$$(kubectl get pods -l app=expressops -n $(K8S_NAMESPACE) -o jsonpath="{.items[0].metadata.name}"); \
	if [ -n "$$POD" ]; then \
		kubectl port-forward svc/expressops $(HOST_PORT):80 -n $(K8S_NAMESPACE); \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-delete:
	@echo "🗑️ Removing ExpressOps from Kubernetes..."
	kubectl delete -f k8s/service.yaml --ignore-not-found
	kubectl delete -f k8s/deployment.yaml --ignore-not-found
	kubectl delete -f k8s/secrets.yaml --ignore-not-found
	kubectl delete -f k8s/configmap.yaml --ignore-not-found
	@echo "✅ ExpressOps removed from Kubernetes"

.DEFAULT_GOAL := help