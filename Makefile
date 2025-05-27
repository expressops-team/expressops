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
RESET  = \033[0m

# ---------------------------------------------
# 🔧 Configurable variables
IMAGE_NAME        ?= expressops
CONTAINER_NAME    ?= expressops-app
NEW_TAG :=
HOST_PORT         ?= 8080
SERVER_PORT       ?= 8080
SERVER_ADDRESS    ?= 0.0.0.0
TIMEOUT_SECONDS   ?= 4
LOG_LEVEL         ?= info
LOG_FORMAT        ?= text
SLACK_WEBHOOK_URL ?=
CONFIG_PATH       ?= docs/samples/config.yaml
CONFIG_MOUNT_PATH ?= /app/config.yaml
HELM_CHART_DIR    ?= k3s/expressops-chart
HELM_VALUES_FILE  ?= $(HELM_CHART_DIR)/values.yaml
DOCKER_TAG_FILE    = .docker_tag

# ---------------------------------------------
# 📜 Goals (.PHONY)
.PHONY: build run docker-build docker-run docker-run-build docker-clean \
        docker-run-sre2 docker-push helm-deploy release help

# ---------------------------------------------

# 🔨 Build plugins and application locally
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

# ---------------------------------------------
# ▶️ Run
run: build
	@echo "🚀 Starting ExpressOps"
	./expressops -config $(CONFIG_PATH)

# ---------------------------------------------
# 🐳 Auto-versioned images Docker build
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
		-t $(IMAGE_NAME):$$NEW_TAG .; \
	echo "✅ Image built: $(IMAGE_NAME):$$NEW_TAG"; \
	\
	echo "🔄 Updating Helm values file: $(HELM_VALUES_FILE) with new tag: $$NEW_TAG"; \
	yq e '.image.tag = "'$$NEW_TAG'"' -i $(HELM_VALUES_FILE); \
	echo "✅ Helm values file updated."

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

docker-push:
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || { echo "$(RED)Error: .docker_tag file not found. Run 'make docker-build' first.$(RESET)"; exit 1; }); \
	echo " Pushing image $(IMAGE_NAME):$$NEW_TAG to Docker Hub..."; \
	docker tag $(IMAGE_NAME):$$NEW_TAG expressopsfreepik/$(IMAGE_NAME):$$NEW_TAG; \
	docker push expressopsfreepik/$(IMAGE_NAME):$$NEW_TAG; \
	echo "✅ Image pushed: expressopsfreepik/$(IMAGE_NAME):$$NEW_TAG"

# ---------------------------------------------
# 🚀 Deploy Helm chart to namespace expressops-dev
helm-deploy:
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || { echo "$(RED)Error: .docker_tag file not found. Ensure image is built or set tag manually.$(RESET)"; exit 1; }); \
	echo "🚀 Deploying Helm chart expressops with image tag $$NEW_TAG to namespace expressops-dev..."; \
	helm upgrade --install expressops $(HELM_CHART_DIR) -n expressops-dev --create-namespace \
		--set image.tag=$$NEW_TAG # Opcional: pasar como --set si no quieres modificar values.yaml
		# O si values.yaml ya está actualizado:
		# helm upgrade --install expressops $(HELM_CHART_DIR) -n expressops-dev --create-namespace

# ---------------------------------------------
# 📦 Release end-to-end
release: docker-build docker-push helm-deploy
	@echo "✅ Release process completed."


# ---------------------------------------------
# 📖 Help
help:
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
	@echo ""
	@echo "  $(BLUE)Helm Deployment Workflow:$(RESET)"
	@echo "    make helm-deploy            - Deploy/Upgrade Helm chart using tag from values.yaml"
	@echo ""
	@echo "  $(BLUE)Combined Release Workflow:$(RESET)"
	@echo "    make release                - Full cycle: docker-build, docker-push, helm-deploy"
	@echo "================================================"
	@echo ""
	@echo "Configurable variables (current values):"
	@echo "  IMAGE_NAME                = $(IMAGE_NAME)"
	@echo "  CONTAINER_NAME            = $(CONTAINER_NAME)"
	@echo "  HOST_PORT                 = $(HOST_PORT)"
	@echo "  SERVER_PORT               = $(SERVER_PORT)"
	@echo "  SERVER_ADDRESS            = $(SERVER_ADDRESS)"
	@echo "  TIMEOUT_SECONDS           = $(TIMEOUT_SECONDS)"
	@echo "  LOG_LEVEL                 = $(LOG_LEVEL)"
	@echo "  LOG_FORMAT                = $(LOG_FORMAT)"
	@echo "  CONFIG_PATH               = $(CONFIG_PATH)"
	@echo "  CONFIG_MOUNT_PATH         = $(CONFIG_MOUNT_PATH)"
	@echo "  HELM_CHART_DIR            = $(HELM_CHART_DIR)"
	@echo "  HELM_VALUES_FILE          = $(HELM_VALUES_FILE)"
