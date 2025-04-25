# docker run -d --name expressops-demo -p 8080:8080 expressops:1.0.0  <-- for testing
#      make docker-build   //    make docker-run
# make build-plugins // make run // make clean // make help // make docker-build // make docker-run

# make run <===
GREEN = \033[32m
RED = \033[31m
BLUE = \033[34m
YELLOW = \033[33m
BOLD = \033[1m
RESET = \033[0m
PRINT = @echo 

# IMAGE REPOSITORY WILL BE CHANGED TO expressopsfreepik/expressops IN THE FUTURE
IMAGE_REPOSITORY ?= davidnull/expressops
IMAGE_TAG ?= 1.0.0 # TODO: vx to make it dynamic
PLUGINS_PATH ?= plugins

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
GCP_SA_KEY_FILE ?= 

.PHONY: build run docker-build docker-push docker-run docker-clean help k8s-deploy k8s-status k8s-logs k8s-delete k8s-port-forward k8s-install-eso helm-install helm-upgrade helm-uninstall helm-template helm-package k8s-apply-fake-clustersecretstore k8s-apply-externalsecret k8s-setup-fake-secrets k8s-verify-secrets helm-install-with-secrets k8s-deploy-with-clustersecretstore k8s-deploy-with-gcp-secretstore helm-install-with-gcp-secrets

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

# Build Docker image
docker-build:
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
	docker tag $(IMAGE_REPOSITORY):$(IMAGE_TAG) davidnull/expressops:$(IMAGE_TAG)
	@echo "✅ Image tagged: davidnull/expressops:$(IMAGE_TAG)"

# Push Docker image to Docker Hub
docker-push: docker-build
	@echo "⬆️ Pushing image to Docker Hub..."
	docker login
	docker push davidnull/expressops:$(IMAGE_TAG)
	@echo "✅ Image pushed to Docker Hub"

# Run Docker container
docker-run:
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

# Run Docker container with build
docker-run-build: docker-build docker-run

# Clean Docker resources
docker-clean:
	@echo "🧹 Cleaning Docker resources..."
	-docker stop $(CONTAINER_NAME) 2>/dev/null || true
	-docker rm $(CONTAINER_NAME) 2>/dev/null || true
	-docker rmi $(IMAGE_REPOSITORY):$(IMAGE_TAG) 2>/dev/null || true
	@echo "🗑 Removing <none> images..."
	-docker rmi $$(docker images -f "dangling=true" -q) 2>/dev/null || true
	@echo "✅ Cleanup completed"

# how this works
help:
	@echo
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo "$(YELLOW)===================$(BOLD)$(BLUE)IMPORTANT READ THE COMMENTS IN THE CODE$(RESET)$(YELLOW)=======================$(RESET)"
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo
	@echo "$(YELLOW)=============================$(BOLD)$(BLUE)MOST USEFUL COMMANDS$(RESET)$(YELLOW)================================$(RESET)"
	@echo "$(GREEN)$(BOLD)  make helm-install-with-secrets $(RESET)- Install ExpressOps with ClusterSecretStore"
	@echo "$(GREEN)$(BOLD)  make k8s-deploy-with-clustersecretstore $(RESET)- Deploy ExpressOps with ClusterSecretStore"
	@echo "$(GREEN)$(BOLD)  make k8s-deploy-with-gcp-secretstore $(RESET)- Deploy ExpressOps with GCP Secret Manager"
	@echo "$(GREEN)$(BOLD)  make helm-install-with-gcp-secrets $(RESET)- Install ExpressOps with GCP Secret Manager via Helm"
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo
	@echo "$(BLUE)Available commands:$(RESET)"
	@echo "$(GREEN)  make help          $(RESET)- Show this help"
	@echo "$(GREEN)  make build         $(RESET)- Build plugins and application"
	@echo "$(GREEN)  make run           $(RESET)- Run application locally"
	@echo "$(GREEN)  make docker-build  $(RESET)- Build Docker image"
	@echo "$(GREEN)  make docker-push   $(RESET)- Build, tag and push Docker image to Docker Hub"
	@echo "$(GREEN)  make docker-run    $(RESET)- Run container"

	@echo "$(GREEN)  make k8s-install-eso $(RED)- Required before first deployment$(RESET)"
	@echo "$(GREEN)  make k8s-deploy    $(RESET)- Deploy to Kubernetes"
	@echo "$(GREEN)  make k8s-status    $(RESET)- Check Kubernetes deployment status"
	@echo "$(GREEN)  make k8s-logs      $(RESET)- View Kubernetes logs"
	@echo "$(GREEN)  make k8s-port-forward $(RESET)- Port forward to access the application"
	@echo "$(GREEN)  make k8s-delete    $(RESET)- Delete Kubernetes deployment"
	
	@echo "$(GREEN)  make helm-install  $(RESET)- Install ExpressOps using Helm chart"
	@echo "$(GREEN)  make helm-upgrade  $(RESET)- Upgrade existing Helm deployment"
	@echo "$(GREEN)  make helm-uninstall$(RESET)- Uninstall Helm deployment"
	@echo "$(GREEN)  make helm-template $(RESET)- View Helm templates without installing"
	@echo "$(GREEN)  make helm-package  $(RESET)- Package Helm chart into a .tgz file"

	@echo
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo
	@echo "$(BLUE)Configurable variables (with our current values):$(RESET)"
	@echo "$(GREEN)  IMAGE_REPOSITORY $(RESET)= $(IMAGE_REPOSITORY)"
	@echo "$(GREEN)  IMAGE_TAG        $(RESET)= $(IMAGE_TAG)"
	@echo "$(GREEN)  PLUGINS_PATH     $(RESET)= $(PLUGINS_PATH)"
	@echo "$(GREEN)  CONTAINER_NAME   $(RESET)= $(CONTAINER_NAME)"
	@echo "$(GREEN)  HOST_PORT        $(RESET)= $(HOST_PORT)"
	@echo "$(GREEN)  SERVER_PORT      $(RESET)= $(SERVER_PORT)"
	@echo "$(GREEN)  SERVER_ADDRESS   $(RESET)= $(SERVER_ADDRESS)"
	@echo "$(GREEN)  TIMEOUT_SECONDS  $(RESET)= $(TIMEOUT_SECONDS)"
	@echo "$(GREEN)  LOG_LEVEL        $(RESET)= $(LOG_LEVEL)"
	@echo "$(GREEN)  LOG_FORMAT       $(RESET)= $(LOG_FORMAT)"
	@echo "$(GREEN)  SLACK_WEBHOOK_URL $(RESET)= ..."
	@echo "$(GREEN)  CONFIG_PATH      $(RESET)= $(CONFIG_PATH)"
	@echo "$(GREEN)  CONFIG_MOUNT_PATH $(RESET)= $(CONFIG_MOUNT_PATH)"
	@echo "$(GREEN)  K8S_NAMESPACE    $(RESET)= $(K8S_NAMESPACE)"
	@echo "$(GREEN)  GCP_SA_KEY_FILE  $(RESET)= $(GCP_SA_KEY_FILE) (Path to service account key json file)"
	@echo
	@echo "$(YELLOW)=================================================================================$(RESET)"
	@echo "$(YELLOW)=======================$(BOLD)$(BLUE)GOOGLE CLOUD SECRET MANAGER$(RESET)$(YELLOW)=========================$(RESET)"
	@echo "$(BLUE)Service Account:$(RESET) expressops-external-secrets@fc-it-school-2025.iam.gserviceaccount.com"
	@echo "$(BLUE)Secret Path:$(RESET) projects/88527591198/secrets/slack-webhook"
	@echo "$(YELLOW)=================================================================================$(RESET)"

# Install External Secrets Operator
# Before deploying:
# 1. Set the SLACK_WEBHOOK_URL environment variable (required):
#    export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/REAL/TOKEN"
# - In case there was one before, you can delete it with:
# ==>  	kubectl delete secret expressops-secrets
k8s-install-eso:
	@echo "🔄 Installing External Secrets Operator..."
	@helm repo add external-secrets https://charts.external-secrets.io
	@helm repo update
	@helm install external-secrets external-secrets/external-secrets \
		--namespace external-secrets \
		--create-namespace \
		--set installCRDs=true
	@echo "✅ External Secrets Operator installed"
	@echo "⏳ Wait for operator to be ready..."
	@kubectl wait --for=condition=available --timeout=90s deployment/external-secrets -n external-secrets || echo "⚠️ Timeout waiting for ESO to be ready"

# Helm install usando ClusterSecretStore
helm-install-with-secrets:
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "$(RED)Error: SLACK_WEBHOOK_URL environment variable is required$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)🚀 Instalando ExpressOps con Helm usando ClusterSecretStore...$(RESET)"
	@echo "$(BLUE)Desplegando en namespace: $(K8S_NAMESPACE)$(RESET)"
	helm upgrade --install expressops ./helm \
		--namespace $(K8S_NAMESPACE) \
		--set clusterSecretStore.webhookUrl="$(SLACK_WEBHOOK_URL)"
	@echo "$(GREEN)✅ ExpressOps instalado correctamente con secretos$(RESET)"
	@echo "$(YELLOW)Para acceder a la aplicación:$(RESET) make k8s-port-forward"

# Usar ClusterSecretStore en deploy normal (no Helm)
k8s-deploy-with-clustersecretstore:
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "$(RED)Error: SLACK_WEBHOOK_URL environment variable is required$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)🔄 Preparando ClusterSecretStore con webhook URL...$(RESET)"
	@sed -e "s|YOUR_SLACK_WEBHOOK_URL_HERE|$(SLACK_WEBHOOK_URL)|g" \
		k8s/secrets/template-clustersecretstore.yaml > k8s/secrets/fake-clustersecretstore.yaml
	
	@echo "$(BLUE)🔄 Desplegando ExpressOps a Kubernetes...$(RESET)"
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/expressops-env-config.yaml 
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/secrets/fake-clustersecretstore.yaml
	kubectl apply -f k8s/secrets/slack-externalsecret.yaml
	kubectl apply -f k8s/service.yaml
	@echo "$(GREEN)✅ ExpressOps desplegado con ClusterSecretStore$(RESET)"
	@echo "$(YELLOW)Para acceder a la aplicación:$(RESET) make k8s-port-forward"

# Deploy to Kubernetes with GCP Secret Manager
k8s-deploy-with-gcp-secretstore:
	@echo "$(BLUE)🔄 Deploying ExpressOps with GCP Secret Manager...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: GCP_SA_KEY_FILE environment variable must point to a service account key file$(RESET)"; \
		echo "Example usage: GCP_SA_KEY_FILE=/path/to/service-account.json make k8s-deploy-with-gcp-secretstore"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)🔄 Creating GCP service account secret...$(RESET)"
	kubectl create secret generic gcp-secret-creds --from-file=sa.json=$(GCP_SA_KEY_FILE) --dry-run=client -o yaml | kubectl apply -f -
	
	@echo "$(BLUE)🔄 Deploying Kubernetes resources...$(RESET)"
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/expressops-env-config.yaml
	kubectl apply -f k8s/secrets/gcp-clustersecretstore.yaml
	kubectl apply -f k8s/secrets/expressops-externalsecret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	
	@echo "$(GREEN)✅ ExpressOps deployed with GCP Secret Manager$(RESET)"
	@echo "$(YELLOW)For accessing the application:$(RESET) make k8s-port-forward"

# Deploy to Helm with GCP Secret Manager
helm-install-with-gcp-secrets:
	@echo "$(BLUE)🚀 Installing ExpressOps with Helm using GCP Secret Manager...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: GCP_SA_KEY_FILE environment variable must point to a service account key file$(RESET)"; \
		echo "Example usage: GCP_SA_KEY_FILE=/path/to/service-account.json make helm-install-with-gcp-secrets"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)Deploying in namespace: $(K8S_NAMESPACE)$(RESET)"
	helm upgrade --install expressops ./helm \
		--namespace $(K8S_NAMESPACE) \
		--set gcpSecretManager.enabled=true \
		--set gcpSecretManager.projectID=fc-it-school-2025 \
		--set externalSecrets.remoteRef.key=projects/88527591198/secrets/slack-webhook \
		--set-file gcpSecretManager.serviceAccountKey=$(GCP_SA_KEY_FILE)
	
	@echo "$(GREEN)✅ ExpressOps installed correctly with GCP Secret Manager$(RESET)"
	@echo "$(YELLOW)For accessing the application:$(RESET) make k8s-port-forward"

# Kubernetes Deployment
# Before deploying:
# 1. Set the SLACK_WEBHOOK_URL environment variable (required):
#    export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/REAL/TOKEN"
# 2. Connect to Kubernetes with the SSH tunnel:
#    gcloud compute ssh --zone "europe-west1-d" "it-school-2025-1" --tunnel-through-iap --project "fc-it-school-2025" --ssh-flag "-N -L 6443:127.0.0.1:6443"
# 3. Build and push the image to Docker Hub (optional):
#    make docker-push
k8s-deploy:
	@echo "🔄 Deploying ExpressOps to Kubernetes..."
	@echo "📦 Applying Kubernetes resources..."
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/expressops-env-config.yaml 
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/secrets/fake-secretstore.yaml
	kubectl apply -f k8s/secrets/slack-externalsecret.yaml
	kubectl apply -f k8s/service.yaml
	@echo "⏳ Waiting for External Secret to sync (15s)..." #to give time for the secret to be created
	@sleep 15
	@if kubectl get secret expressops-secrets >/dev/null 2>&1; then \
		echo "✅ Secret 'expressops-secrets' created successfully"; \
	else \
		echo "⚠️ Secret 'expressops-secrets' not created yet. You may need to install External Secrets Operator."; \
		echo "   Run: make k8s-install-eso"; \
	fi
	@echo "✅ ExpressOps deployed to Kubernetes"
	@echo "🔍 Verify status with: make k8s-status"
	@echo "🌐 Access the application with: make k8s-port-forward"

k8s-status:
	@echo "🔍 Checking ExpressOps deployment status:"
	@echo "\n📊 Pods status:"
	@POD_NAME=$$(kubectl get pods -n $(K8S_NAMESPACE) | grep "^expressops-" | awk '{print $$1}' | head -1); \
	if [ -n "$$POD_NAME" ]; then \
		kubectl get pod $$POD_NAME -n $(K8S_NAMESPACE); \
		echo "\n📋 Pod logs:"; \
		kubectl logs $$POD_NAME -n $(K8S_NAMESPACE) --tail=10; \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi
	@echo "\n🌐 Service status:"
	@kubectl get svc expressops -n $(K8S_NAMESPACE)

k8s-logs:
	@echo "📃 ExpressOps logs:"
	@POD_NAME=$$(kubectl get pods -n $(K8S_NAMESPACE) | grep "^expressops-" | awk '{print $$1}' | head -1); \
	if [ -n "$$POD_NAME" ]; then \
		kubectl logs $$POD_NAME -n $(K8S_NAMESPACE) --tail=100; \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-port-forward:
	@echo "🔄 Setting up port forwarding for ExpressOps service..."
	@echo "🌐 Access the application at http://localhost:$(HOST_PORT)"
	@POD_NAME=$$(kubectl get pods -n $(K8S_NAMESPACE) | grep "^expressops-" | awk '{print $$1}' | head -1); \
	if [ -n "$$POD_NAME" ]; then \
		echo "🔌 Forwarding to pod: $$POD_NAME"; \
		kubectl port-forward pod/$$POD_NAME $(HOST_PORT):$(SERVER_PORT) -n $(K8S_NAMESPACE); \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-delete:
	@echo "🗑️ Deleting ExpressOps from Kubernetes..."
	kubectl delete -f k8s/service.yaml --ignore-not-found
	kubectl delete -f k8s/deployment.yaml --ignore-not-found
	kubectl delete -f k8s/secrets/slack-externalsecret.yaml --ignore-not-found
	kubectl delete -f k8s/secrets/fake-secretstore.yaml --ignore-not-found				
	kubectl delete -f k8s/configmap.yaml --ignore-not-found
	kubectl delete -f k8s/expressops-env-config.yaml --ignore-not-found
	@echo "✅ ExpressOps deleted from Kubernetes"

# Helm Commands
helm-install:
	@echo "🚀 Instalando ExpressOps con Helm..."
	@echo "$(BLUE)Desplegando en namespace: $(K8S_NAMESPACE)$(RESET)"
	helm install expressops ./helm --namespace $(K8S_NAMESPACE)
	@echo "✅ Helm chart instalado correctamente"

helm-upgrade:
	@echo "🔄 Actualizando ExpressOps con Helm..."
	helm upgrade expressops ./helm --namespace $(K8S_NAMESPACE)
	@echo "✅ Helm chart actualizado correctamente"

helm-uninstall:
	@echo "🗑️ Desinstalando ExpressOps de Helm..."
	helm uninstall expressops --namespace $(K8S_NAMESPACE)
	@echo "✅ Helm chart desinstalado correctamente"

helm-template:
	@echo "👀 Visualizando plantillas renderizadas..."
	helm template expressops ./helm --namespace $(K8S_NAMESPACE)

helm-package:
	@echo "📦 Empaquetando Helm chart..."
	helm package ./helm
	@echo "✅ Chart empaquetado. Listo para distribuir."

# Secret Management Targets
k8s-apply-fake-clustersecretstore:
	kubectl apply -f k8s/secrets/fake-clustersecretstore.yaml
	@echo "Fake ClusterSecretStore applied"

k8s-apply-externalsecret:
	kubectl apply -f k8s/secrets/slack-externalsecret.yaml
	@echo "ExternalSecret applied"

k8s-setup-fake-secrets: k8s-apply-fake-clustersecretstore k8s-apply-externalsecret
	@echo "Fake secret management setup complete"
	@echo "Wait a moment for the ExternalSecret to create the actual Kubernetes secret"
	sleep 5
	kubectl get secret expressops-slack-secret

# Verify secrets are working
k8s-verify-secrets:
	@echo "Verifying that the secret was created:"
	kubectl get secret expressops-slack-secret
	@echo "Little Reminder: The secret's content is controlled by the External Secrets Operator ;D"

.DEFAULT_GOAL := help