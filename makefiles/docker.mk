# Docker operations
.PHONY: docker-build docker-push docker-run docker-run-build docker-clean

## Docker operations for building and running containers

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg SERVER_ADDRESS=$(SERVER_ADDRESS) \
		--build-arg TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		--build-arg LOG_LEVEL=$(LOG_LEVEL) \
		--build-arg LOG_FORMAT=$(LOG_FORMAT) \
		--build-arg CONFIG_PATH=$(CONFIG_MOUNT_PATH) \
		-t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .
	@echo "✅ Image built: $(IMAGE_REPOSITORY):$(IMAGE_TAG)"
	@echo "🏷️ Tagging image for Docker Hub..."
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_TAG) $(IMAGE_REPOSITORY):$(IMAGE_TAG)
	@echo "✅ Image tagged: $(IMAGE_REPOSITORY):$(IMAGE_TAG)"

docker-push: docker-build ## Build and push Docker image to Docker Hub
	@echo "⬆️ Pushing image to Docker Hub..."
	docker login
	docker push $(IMAGE_REPOSITORY):$(IMAGE_TAG)
	@echo "✅ Image pushed to Docker Hub"

docker-run: ## Run Docker container
	@echo "🚀 Starting container..."
	@echo "📌 Application available at http://localhost:$(HOST_PORT)"
	docker run --name $(CONTAINER_NAME) \
		-p $(HOST_PORT):$(SERVER_PORT) \
		-e SERVER_PORT=$(SERVER_PORT) \
		-e SERVER_ADDRESS=$(SERVER_ADDRESS) \
		-e TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		-e LOG_LEVEL=$(LOG_LEVEL) \
		-e LOG_FORMAT=$(LOG_FORMAT) \
		-e SLACK_WEBHOOK_URL=$(SLACK_WEBHOOK_URL) \
		-v $(PWD)/$(CONFIG_PATH):$(CONFIG_MOUNT_PATH) \
		--rm $(IMAGE_REPOSITORY):$(IMAGE_TAG)

docker-run-build: docker-build docker-run ## Build and run Docker container

docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_REPOSITORY):$(IMAGE_TAG) 2>/dev/null || true
	@echo "🗑 Removing <none> images..."
	-docker rmi $$(docker images -f "dangling=true" -q) 2>/dev/null || true
	@echo "✅ Cleanup completed" 