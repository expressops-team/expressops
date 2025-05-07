# Helm operations
.PHONY: helm-install helm-upgrade helm-uninstall helm-template helm-package helm-install-with-secrets helm-install-with-gcp-secrets

## Helm chart operations for deployment and management

helm-install: ## Install ExpressOps using Helm chart
	@echo "🚀 Instalando ExpressOps con Helm..."
	@echo "$(BLUE)Desplegando en namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm install expressops ./helm --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart instalado correctamente"

helm-upgrade: ## Upgrade existing Helm deployment
	@echo "🔄 Actualizando ExpressOps con Helm..."
	-helm upgrade expressops ./helm --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart actualizado correctamente"

helm-diff: ## Diff Helm deployment
	@echo "🔄 Diferencias en Helm chart..."
	-helm diff upgrade expressops ./helm --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret
	@echo "✅ Helm chart actualizado correctamente"

helm-uninstall: ## Uninstall Helm deployment
	@echo "🗑️ Desinstalando ExpressOps de Helm..."
	-helm uninstall expressops --namespace $(K8S_NAMESPACE)
	@echo "✅ Helm chart desinstalado correctamente"

helm-template: ## View Helm templates without installing
	@echo "👀 Visualizando plantillas renderizadas..."
	-helm template expressops ./helm --namespace $(K8S_NAMESPACE) --set secrets.secretName=expressops-slack-secret

helm-package: ## Package Helm chart into a .tgz file
	@echo "📦 Empaquetando Helm chart..."
	-helm package ./helm
	@echo "✅ Chart empaquetado. Listo para distribuir."

helm-install-with-secrets: ## Install ExpressOps with ClusterSecretStore (legacy)
	@if [ -z "$(SLACK_WEBHOOK_URL)" ]; then \
		echo "$(RED)Error: SLACK_WEBHOOK_URL environment variable is required$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)🚀 Instalando ExpressOps con Helm usando ClusterSecretStore...$(RESET)"
	@echo "$(BLUE)Desplegando en namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm upgrade --install expressops ./helm \
		--namespace $(K8S_NAMESPACE) \
		--set clusterSecretStore.webhookUrl="$(SLACK_WEBHOOK_URL)" \
		--set secrets.secretName=expressops-slack-secret
	@echo "$(GREEN)✅ ExpressOps instalado correctamente con secretos$(RESET)"
	@echo "$(YELLOW)Para acceder a la aplicación:$(RESET) make k8s-port-forward"

helm-install-with-gcp-secrets: ## Install ExpressOps with GCP Secret Manager via Helm
	@echo "$(BLUE)🚀 Installing ExpressOps with Helm using GCP Secret Manager...$(RESET)"
	@if [ ! -f "$(GCP_SA_KEY_FILE)" ]; then \
		echo "$(RED)Error: Service account key file $(GCP_SA_KEY_FILE) not found$(RESET)"; \
		echo "Please ensure key.json is present in the project root directory"; \
		exit 1; \
	fi
	
	@echo "$(BLUE)Deploying in namespace: $(K8S_NAMESPACE)$(RESET)"
	-helm upgrade --install expressops ./helm \
		--namespace $(K8S_NAMESPACE) \
		--set gcpSecretManager.enabled=true \
		--set gcpSecretManager.projectID=fc-it-school-2025 \
		--set externalSecrets.remoteRef.key=projects/88527591198/secrets/slack-webhook \
		--set secrets.secretName=expressops-slack-secret \
		--set-file gcpSecretManager.serviceAccountKey=$(GCP_SA_KEY_FILE)
	
	@echo "$(GREEN)✅ ExpressOps installed correctly with GCP Secret Manager$(RESET)"
	@echo "$(YELLOW)For accessing the application:$(RESET) make k8s-port-forward" 