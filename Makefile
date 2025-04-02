# make build-plugins // make run // make clean // make help

# make run <===
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
RESET = \033[0m
PRINT = @echo 

.PHONY: build-plugins run clean help

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

help:
	@echo "make build-plugins - Build plugins"
	@echo "make run          - Build and run"
	@echo "make clean        - Clean .so files"
	@echo "make help         - Help"

.DEFAULT_GOAL := build-plugins