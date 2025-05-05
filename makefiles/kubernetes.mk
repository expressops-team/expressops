# Kubernetes operations
.PHONY: k8s-deploy k8s-status k8s-logs k8s-delete k8s-port-forward k8s-install-eso k8s-apply-gcp-secretstore k8s-apply-externalsecret k8s-setup-gcp-secrets k8s-verify-secrets k8s-deploy-with-clustersecretstore k8s-deploy-with-gcp-secretstore setup-with-gcp-credentials

## Kubernetes deployment and management operations

k8s-install-eso: ## Install External Secrets Operator (required before first deployment)
	@echo "🔄 Installing External Secrets Operator..."
	@helm repo add external-secrets https://charts.external-secrets.io || true
	@helm repo update || true
	@if helm list -n external-secrets | grep -q "external-secrets"; then \
		echo "$(YELLOW)⚠️ External Secrets Operator already installed. Skipping installation.$(RESET)"; \
	else \
		echo "$(BLUE)Installing External Secrets Operator...$(RESET)"; \
		helm install external-secrets external-secrets/external-secrets \
			--namespace external-secrets \
			--create-namespace \
			--set installCRDs=true || true; \
	fi
	@echo "✅ External Secrets Operator setup completed"
	@echo "⏳ Wait for operator to be ready..."
	@kubectl wait --for=condition=available --timeout=90s deployment/external-secrets -n external-secrets 2>/dev/null || echo "⚠️ Timeout waiting for ESO to be ready"

k8s-deploy: ## Deploy application to Kubernetes
	@echo "🔄 Deploying ExpressOps to Kubernetes..."
	@echo "📦 Applying Kubernetes resources..."
	-kubectl apply -f k8s/configmap.yaml || true
	-kubectl apply -f k8s/expressops-env-config.yaml || true
	-kubectl apply -f k8s/deployment.yaml || true
	-kubectl apply -f k8s/secrets/gcp-clustersecretstore.yaml || true
	-kubectl apply -f k8s/secrets/expressops-externalsecret.yaml || true
	-kubectl apply -f k8s/service.yaml || true
	@echo "⏳ Waiting for External Secret to sync (15s)..." #to give time for the secret to be created
	@sleep 15
	@if kubectl get secret expressops-slack-secret >/dev/null 2>&1; then \
		echo "✅ Secret 'expressops-slack-secret' created successfully"; \
	else \
		echo "⚠️ Secret 'expressops-slack-secret' not created yet. You may need to install External Secrets Operator."; \
		echo "   Run: make k8s-install-eso"; \
	fi
	@echo "✅ ExpressOps deployed to Kubernetes"
	@echo "🔍 Verify status with: make k8s-status"
	@echo "🌐 Access the application with: make k8s-port-forward"

k8s-deploy-with-clustersecretstore: ## Deploy using ClusterSecretStore (legacy)
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "$(RED)Error: SLACK_WEBHOOK_URL environment variable is required$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)🔄 Preparando y desplegando ExpressOps a Kubernetes...$(RESET)"
	-kubectl apply -f k8s/configmap.yaml || true
	-kubectl apply -f k8s/expressops-env-config.yaml || true
	-kubectl apply -f k8s/deployment.yaml || true
	-kubectl apply -f k8s/secrets/gcp-clustersecretstore.yaml || true
	-kubectl apply -f k8s/secrets/expressops-externalsecret.yaml || true
	-kubectl apply -f k8s/service.yaml || true
	@echo "$(GREEN)✅ ExpressOps desplegado con ClusterSecretStore$(RESET)"
	@echo "$(YELLOW)Para acceder a la aplicación:$(RESET) make k8s-port-forward"

k8s-deploy-with-gcp-secretstore: ## Deploy with GCP Secret Manager
	@echo "$(BLUE)🔄 Deploying ExpressOps with GCP Secret Manager...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: Service account key file $(GCP_SA_KEY_FILE) not found$(RESET)"; \
		echo "Please ensure key.json is present in the project root directory"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)🔄 Creating GCP service account secret...$(RESET)"
	-kubectl create secret generic expressops-gcp-sa --from-file=sa.json=$(GCP_SA_KEY_FILE) --dry-run=client -o yaml | kubectl apply -f - || true
	
	@echo "$(BLUE)🔄 Deploying Kubernetes resources...$(RESET)"
	-kubectl apply -f k8s/configmap.yaml || true
	-kubectl apply -f k8s/expressops-env-config.yaml || true
	-kubectl apply -f k8s/secrets/gcp-clustersecretstore.yaml || true
	-kubectl apply -f k8s/secrets/expressops-externalsecret.yaml || true
	-kubectl apply -f k8s/deployment.yaml || true
	-kubectl apply -f k8s/service.yaml || true
	
	@echo "$(GREEN)✅ ExpressOps deployed with GCP Secret Manager$(RESET)"
	@echo "$(YELLOW)For accessing the application:$(RESET) make k8s-port-forward"

k8s-status: ## Check Kubernetes deployment status
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

k8s-logs: ## View Kubernetes logs
	@echo "📃 ExpressOps logs:"
	@POD_NAME=$$(kubectl get pods -n $(K8S_NAMESPACE) | grep "^expressops-" | awk '{print $$1}' | head -1); \
	if [ -n "$$POD_NAME" ]; then \
		kubectl logs $$POD_NAME -n $(K8S_NAMESPACE) --tail=100; \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-port-forward: ## Port forward to access the application
	@echo "🔄 Setting up port forwarding for ExpressOps service..."
	@echo "🌐 Access the application at http://localhost:$(HOST_PORT)"
	@POD_NAME=$$(kubectl get pods -n $(K8S_NAMESPACE) | grep "^expressops-" | awk '{print $$1}' | head -1); \
	if [ -n "$$POD_NAME" ]; then \
		echo "🔌 Forwarding to pod: $$POD_NAME"; \
		kubectl port-forward pod/$$POD_NAME $(HOST_PORT):$(SERVER_PORT) -n $(K8S_NAMESPACE); \
	else \
		echo "❌ No ExpressOps pods found"; \
	fi

k8s-delete: ## Delete Kubernetes deployment
	@echo "🗑️ Deleting ExpressOps from Kubernetes..."
	-kubectl delete -f k8s/service.yaml --ignore-not-found || true
	-kubectl delete -f k8s/deployment.yaml --ignore-not-found || true
	-kubectl delete -f k8s/secrets/expressops-externalsecret.yaml --ignore-not-found || true
	-kubectl delete -f k8s/secrets/gcp-clustersecretstore.yaml --ignore-not-found || true
	-kubectl delete -f k8s/configmap.yaml --ignore-not-found || true
	-kubectl delete -f k8s/expressops-env-config.yaml --ignore-not-found || true
	@echo "✅ ExpressOps deleted from Kubernetes"

# Secret Management
k8s-apply-gcp-secretstore: ## Apply GCP ClusterSecretStore
	-kubectl apply -f k8s/secrets/gcp-clustersecretstore.yaml || true
	@echo "GCP ClusterSecretStore applied"

k8s-apply-externalsecret: ## Apply ExternalSecret
	-kubectl apply -f k8s/secrets/expressops-externalsecret.yaml || true
	@echo "ExternalSecret applied"

k8s-setup-gcp-secrets: k8s-apply-gcp-secretstore k8s-apply-externalsecret ## Setup GCP secrets
	@echo "GCP secret management setup complete"
	@echo "Wait a moment for the ExternalSecret to create the actual Kubernetes secret"
	sleep 5
	-kubectl get secret expressops-slack-secret || true

k8s-verify-secrets: ## Verify secrets are working
	@echo "Verifying that the secret was created:"
	-kubectl get secret expressops-slack-secret || true
	@echo "Little Reminder: The secret's content is controlled by the External Secrets Operator ;D"

setup-with-gcp-credentials: ## Setup complete environment with GCP credentials
	@echo "$(BLUE)🔄 Setting up ExpressOps with GCP credentials from key.json...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: Service account key file $(GCP_SA_KEY_FILE) not found$(RESET)"; \
		echo "Please ensure key.json is present in the project root directory"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)🔄 Installing External Secrets Operator...$(RESET)"
	@make k8s-install-eso || true
	
	@echo "$(BLUE)🔄 Deploying ExpressOps with GCP secrets...$(RESET)"
	@make helm-install-with-gcp-secrets || true
	
	@echo "$(GREEN)✅ ExpressOps setup completed with GCP credentials$(RESET)"
	@echo "$(YELLOW)For accessing the application:$(RESET) make k8s-port-forward" 