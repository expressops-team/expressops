# docker run -d --name expressops-demo -p 8080:8080 expressops:latest   <-- for testing
#      make docker-build   //    make docker-run
# make build-plugins // make run // make clean // make help // make docker-build // make docker-run

# make run <===
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
RESET = \033[0m
PRINT = @echo 

# Variables configurables (se pueden sobrescribir con variables de entorno)
IMAGE_NAME ?= expressops
CONTAINER_NAME ?= expressops-app
HOST_PORT ?= 8080
SERVER_PORT ?= 8080
SERVER_ADDRESS ?= 0.0.0.0
TIMEOUT_SECONDS ?= 4
LOG_LEVEL ?= info
LOG_FORMAT ?= text

.PHONY: build run docker-build docker-run docker-clean help

# Compilar plugins y la aplicación localmente
build:
	@echo "Compilando plugins..."
	@for dir in $$(find plugins -type f -name "*.go" -exec dirname {} \; | sort -u); do \
		for gofile in $$dir/*.go; do \
			if [ -f "$$gofile" ]; then \
				plugin_name=$$(basename "$$gofile" .go); \
				go build -buildmode=plugin -o "$$dir/$$plugin_name.so" "$$gofile"; \
			fi \
		done \
	done
	@echo "Compilando aplicación..."
	@go build -o expressops ./cmd
	@echo "✅ Compilación completada"

# Ejecutar la aplicación localmente
run: build
	@echo "🚀 Iniciando ExpressOps"
	./expressops -config docs/samples/config.yaml

# Construir imagen Docker
docker-build:
	@echo "🐳 Construyendo imagen Docker..."
	docker build \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg SERVER_ADDRESS=$(SERVER_ADDRESS) \
		--build-arg TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		--build-arg LOG_LEVEL=$(LOG_LEVEL) \
		--build-arg LOG_FORMAT=$(LOG_FORMAT) \
		-t $(IMAGE_NAME):latest .
	@echo "✅ Imagen construida: $(IMAGE_NAME):latest"

# Ejecutar contenedor Docker
docker-run: docker-build
	@echo "🚀 Iniciando contenedor..."
	@echo "📌 Aplicación disponible en http://localhost:$(HOST_PORT)"
	docker run --name $(CONTAINER_NAME) \
		-p $(HOST_PORT):$(SERVER_PORT) \
		-e SERVER_PORT=$(SERVER_PORT) \
		-e SERVER_ADDRESS=$(SERVER_ADDRESS) \
		-e TIMEOUT_SECONDS=$(TIMEOUT_SECONDS) \
		-e LOG_LEVEL=$(LOG_LEVEL) \
		-e LOG_FORMAT=$(LOG_FORMAT) \
		-v $(PWD)/docs/samples:/app/config \
		--rm $(IMAGE_NAME):latest

# Limpiar recursos Docker
docker-clean:
	@echo "🧹 Limpiando recursos Docker..."
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	@echo "✅ Limpieza completada"

# Ayuda
help:
	@echo "Comandos disponibles:"
	@echo "  make build         - Compilar plugins y aplicación"
	@echo "  make run           - Ejecutar aplicación localmente"
	@echo "  make docker-build  - Construir imagen Docker"
	@echo "  make docker-run    - Ejecutar contenedor"
	@echo
	@echo "Variables configurables (actuales):"
	@echo "  IMAGE_NAME       = $(IMAGE_NAME)"
	@echo "  CONTAINER_NAME   = $(CONTAINER_NAME)"
	@echo "  HOST_PORT        = $(HOST_PORT)"
	@echo "  SERVER_PORT      = $(SERVER_PORT)"
	@echo "  SERVER_ADDRESS   = $(SERVER_ADDRESS)"
	@echo "  TIMEOUT_SECONDS  = $(TIMEOUT_SECONDS)"
	@echo "  LOG_LEVEL        = $(LOG_LEVEL)"
	@echo "  LOG_FORMAT       = $(LOG_FORMAT)"

.DEFAULT_GOAL := help