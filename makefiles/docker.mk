# Docker operations
.PHONY: docker-build docker-push docker-run docker-run-build docker-clean

## Docker operations for building and running containers

docker-build: generate-new-tag ## Build Docker image (auto-versioned) and updates Helm values
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || echo "latest"); \
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
	echo "🏷️ Tagging image for Docker Hub..."; \
	docker tag $(IMAGE_NAME):$$NEW_TAG $(IMAGE_REPOSITORY):$$NEW_TAG; \
	$(MAKE) update-helm-values

docker-push: ## Tag and push the last built image to Docker Hub
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || { echo "$(RED)Error: .docker_tag file not found. Run 'make docker-build' first.$(RESET)"; exit 1; }); \
	echo "⬆️ Pushing image to Docker Hub..."; \
	docker push $(IMAGE_REPOSITORY):$$NEW_TAG; \
	echo "✅ Image pushed: $(IMAGE_REPOSITORY):$$NEW_TAG"

docker-run: ## Run Docker container with the last built tag
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || echo "latest"); \
	echo "🚀 Starting container with tag: $$NEW_TAG"; \
	echo "📌 Application available at http://localhost:$(HOST_PORT)"; \
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

docker-run-build: docker-build docker-run ## Build and run Docker container

docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	-rm -f .docker_tag
	@echo "🗑 Removing unused images..."
	@echo "🗑 Removing <none> images..."
	-docker rmi $$(docker images -f "dangling=true" -q) 2>/dev/null || true
	@echo "✅ Cleanup completed" 