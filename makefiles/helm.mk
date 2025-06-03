.PHONY: helm-install helm-upgrade helm-uninstall helm-template helm-package helm-install-with-secrets helm-install-with-gcp-secrets helm-deploy

## Helm chart operations for deployment and management

helm-deploy: ## Deploy Helm chart to namespace with auto-versioned image tag
	@NEW_TAG=$$(cat .docker_tag 2>/dev/null || { echo "$(RED)Error: .docker_tag file not found. Ensure image is built or set tag manually.$(RESET)"; exit 1; }); \
	echo "🚀 Deploying Helm chart expressops with image tag $$NEW_TAG to namespace $(K8S_NAMESPACE)..."; \
	helm upgrade --install expressops $(HELM_CHART_DIR) -n $(K8S_NAMESPACE) --create-namespace \
		--set image.tag=$$NEW_TAG; \
	echo "✅ Helm chart deployed successfully"

helm-install: ## Install ExpressOps using Helm chart
	@echo "🚀 Installing ExpressOps with Helm..."
	@echo "$(BLUE)Deploying in namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm install expressops $(HELM_CHART_DIR) --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart installed successfully"

helm-upgrade: ## Upgrade existing Helm deployment
	@echo "🔄 Upgrading ExpressOps with Helm..."
	-helm upgrade expressops $(HELM_CHART_DIR) --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart upgraded successfully"

helm-diff: ## Diff Helm deployment
	@echo "🔄 Checking differences in Helm chart..."
	-helm diff upgrade expressops $(HELM_CHART_DIR) --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart diff displayed successfully"

helm-uninstall: ## Uninstall Helm deployment
	@echo "🗑️ Uninstalling ExpressOps from Helm..."
	-helm uninstall expressops --namespace $(K8S_NAMESPACE)
	@echo "✅ Helm chart uninstalled successfully"

helm-template: ## View Helm templates without installing
	@echo "👀 Rendering Helm templates..."
	-helm template expressops $(HELM_CHART_DIR) --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret

helm-package: ## Package Helm chart into a .tgz file
	@echo "📦 Packaging Helm chart..."
	-helm package $(HELM_CHART_DIR)
	@echo "✅ Chart packaged. Ready to distribute."

helm-install-with-secrets: ## Install ExpressOps with ClusterSecretStore (legacy)
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "$(RED)Error: SLACK_WEBHOOK_URL environment variable is required$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)🚀 Installing ExpressOps with Helm using ClusterSecretStore...$(RESET)"
	@echo "$(BLUE)Deploying in namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm upgrade --install expressops $(HELM_CHART_DIR) \
		--namespace $(K8S_NAMESPACE) \
		--set clusterSecretStore.webhookUrl="$(SLACK_WEBHOOK_URL)" \
		--set secrets.secretName=expressops-slack-secret
	@echo "$(GREEN)✅ ExpressOps installed successfully with secrets$(RESET)"
	@echo "$(YELLOW)To access the application:$(RESET) make k8s-port-forward"

helm-install-with-gcp-secrets: ## Install ExpressOps with GCP Secret Manager via Helm
	@echo "$(BLUE)🚀 Installing ExpressOps with Helm using GCP Secret Manager...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: Service account key file $(GCP_SA_KEY_FILE) not found$(RESET)"; \
		echo "Please ensure key.json is present in the project root directory"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)Deploying in namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm upgrade --install expressops $(HELM_CHART_DIR) \
		--namespace $(K8S_NAMESPACE) \
		--set gcpSecretManager.enabled=true \
		--set gcpSecretManager.projectID=fc-it-school-2025 \
		--set externalSecrets.remoteRef.key=projects/88527591198/secrets/slack-webhook \
		--set secrets.secretName=expressops-slack-secret \
		--set-file gcpSecretManager.serviceAccountKey=$(GCP_SA_KEY_FILE)
	
	@echo "$(GREEN)✅ ExpressOps installed correctly with GCP Secret Manager$(RESET)"
	@echo "$(YELLOW)To access the application:$(RESET) make k8s-port-forward"
