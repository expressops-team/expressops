# Build operations
.PHONY: build run

## Build operations for ExpressOps application

build: ## Build plugins and application locally
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

run: build ## Run application locally
	@echo "🚀 Starting ExpressOps"
	./expressops -config $(CONFIG_PATH) 
	