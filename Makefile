# docker run -d --name expressops-demo -p 8080:8080 expressops:latest   <-- for testing
# make build-plugins // make run // make clean // make help // make docker-build // make docker-run

# make run <===
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
RESET = \033[0m
PRINT = @echo 

# Docker variables
IMAGE_NAME = expressops
TAG = latest
CONTAINER_NAME = expressops-app
DOCKER_PORT = 8080
HOST_PORT = 8080

.PHONY: build-plugins run clean help docker-build docker-run docker-clean docker-compose

build-plugins:
	$(PRINT) "$(BLUE)Building plugins...$(RESET)"
	@for dir in $(shell find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
		for gofile in $$dir/*.go; do \
			if [ -f "$$gofile" ]; then \
				plugin_name=$$(basename "$$gofile" .go); \
				go build -buildmode=plugin -o "$$dir/$$plugin_name.so" "$$gofile"; \
			fi \
		done \
	done
	$(PRINT) "$(GREEN)✨ Plugins built$(RESET)"

run: build-plugins
	$(PRINT) "$(YELLOW)🎉 Running ExpressOps$(RESET)"
	go run cmd/expressops.go

clean:
	$(PRINT) "$(YELLOW)🧹 Cleaning plugins$(RESET)"
	find plugins -name "*.so" -type f -delete
	$(PRINT) "$(GREEN)✅ Cleanup complete$(RESET)"

docker-build:
	$(PRINT) "$(BLUE)🐳 Building Docker image...$(RESET)"
	docker build -t $(IMAGE_NAME):$(TAG) .
	$(PRINT) "$(GREEN)✅ Docker image built: $(IMAGE_NAME):$(TAG)$(RESET)"

docker-run: docker-build
	$(PRINT) "$(YELLOW)🚀 Running container from image: $(IMAGE_NAME):$(TAG)$(RESET)"
	docker run --name $(CONTAINER_NAME) -p $(HOST_PORT):$(DOCKER_PORT) \
		-e SLACK_WEBHOOK_URL \
		-e SLEEP_DURATION \
		--rm $(IMAGE_NAME):$(TAG)

docker-clean:
	$(PRINT) "$(YELLOW)🧹 Cleaning Docker resources$(RESET)"
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_NAME):$(TAG) 2>/dev/null || true
	$(PRINT) "$(GREEN)✅ Docker cleanup complete$(RESET)"

# Example of a docker-compose target that can be expanded later
docker-compose:
	$(PRINT) "$(BLUE)🔄 Starting services with Docker Compose...$(RESET)"
	docker-compose up --build

help:
	@echo "make build-plugins - Build plugins"
	@echo "make run          - Build and run"
	@echo "make clean        - Clean .so files"
	@echo "make docker-build - Build Docker image"
	@echo "make docker-run   - Run container (builds image if needed)"
	@echo "make docker-clean - Clean Docker containers and images"
	@echo "make docker-compose - Run with Docker Compose"
	@echo "make help         - Help"

.DEFAULT_GOAL := build-plugins