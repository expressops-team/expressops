setup-eso:
	@echo "🚀 Installing/Upgrading External Secrets Operator..."
	helm repo add external-secrets https://charts.external-secrets.io && helm repo update
	helm upgrade --install external-secrets external-secrets/external-secrets \
		-n external-secrets --create-namespace # Ajusta namespace y versión

setup-cluster-secret-store:
	@echo "🚀 Applying ClusterSecretStore..."
	# Asume que tienes un archivo cluster-secretstore.yaml en k3s/ o similar
	kubectl apply -f k3s/cluster-secretstore.yaml
	@echo "ℹ️  ACTION REQUIRED: Ensure the GCP credentials secret (e.g., gcp-creds-for-eso) exists in the namespace specified in ClusterSecretStore (e.g., expressops-dev)."