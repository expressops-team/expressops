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
	go build -buildmode=plugin -o plugins/healthcheck/health_check.so plugins/healthcheck/health_check.go
	go build -buildmode=plugin -o plugins/sleep/sleep_plugin.so plugins/sleep/sleep_plugin.go
	go build -buildmode=plugin -o plugins/slack/slack.so plugins/slack/slack.go
	go build -buildmode=plugin -o plugins/testprint/testprint.so plugins/testprint/testprint.go
	go build -buildmode=plugin -o plugins/formatters/health_alert_formatter.so plugins/formatters/health_alert_formatter.go
	go build -buildmode=plugin -o plugins/clean-disk/clean_disk.so plugins/clean-disk/clean_disk.go
	go build -buildmode=plugin -o plugins/logfilecreator/logfilecreator.so plugins/logfilecreator/logfilecreator.go
	go build -buildmode=plugin -o plugins/logcleaner/logcleaner.so plugins/logcleaner/logcleaner.go
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