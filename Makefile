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

.PHONY: build run docker-build docker-run docker-clean help

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
		-t $(IMAGE_NAME):$$NEW_TAG .; \
	echo "✅ Image built: $(IMAGE_NAME):$$NEW_TAG"


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
	@echo "  make docker-run    - Run container"
	@echo "  make docker-run-sre2 - Run container with SRE2 configuration"
	@echo "  make docker-clean  - Clean Docker resources"
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

.DEFAULT_GOAL := help