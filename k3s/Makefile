APP_NAME=expressops
NAMESPACE=expressops-dev
YAML_DEPLOYMENT=expressops.yaml
YAML_CONFIGMAP=configmap.yaml

# Variables para vendor charts
JUICEFS_NAMESPACE=kube-system
ARGOCD_NAMESPACE=argocd
MONITORING_NAMESPACE=monitoring
EXTERNAL_SECRETS_NAMESPACE=external-secrets

deploy:
	@echo "🚀 Desplegando $(APP_NAME) en $(NAMESPACE)..."
	kubectl apply -f $(YAML_CONFIGMAP)
	kubectl apply -f $(YAML_DEPLOYMENT)
	kubectl rollout status deployment/$(APP_NAME)-deployment -n $(NAMESPACE)

status:
	@echo "📊 Estado de los pods, service y deployment del namespace expressops-dev"
	@kubectl get all -n $(NAMESPACE)
	@echo "\n📊 Estado de los configmaps:"
	@kubectl get configmap -n $(NAMESPACE)

delete:
	@echo "🗑️ Eliminando $(APP_NAME)..."
	kubectl delete -f $(YAML_DEPLOYMENT)
	kubectl delete -f $(YAML_CONFIGMAP)

create-namespace:
	@echo "🔧 Creando namespace $(NAMESPACE) si no existe..."
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

logs:
	@echo "📜 Mostrando logs de $(APP_NAME)..."
	kubectl logs -l app=$(APP_NAME) -n $(NAMESPACE) --tail=100 -f

# Comandos para vendor-charts
install-juicefs:
	@echo "🔧 Instalando JuiceFS CSI Driver..."
	helm dependency update vendor-charts/juicefs-csi-driver
	helm upgrade --install juicefs-csi-driver vendor-charts/juicefs-csi-driver \
		--namespace $(JUICEFS_NAMESPACE) --create-namespace

install-argocd:
	@echo "🔧 Instalando Argo CD..."
	helm dependency update vendor-charts/argo-cd
	helm upgrade --install argo-cd vendor-charts/argo-cd \
		--namespace $(ARGOCD_NAMESPACE) --create-namespace

install-prometheus:
	@echo "🔧 Instalando Prometheus Stack..."
	helm dependency update vendor-charts/prometheus-stack
	helm upgrade --install prometheus-stack vendor-charts/prometheus-stack \
		--namespace $(MONITORING_NAMESPACE) --create-namespace

install-loki:
	@echo "🔧 Instalando Loki..."
	helm dependency update vendor-charts/loki
	helm upgrade --install loki vendor-charts/loki \
		--namespace $(MONITORING_NAMESPACE) --create-namespace

install-fluentbit:
	@echo "🔧 Instalando Fluent Bit..."
	helm dependency update vendor-charts/fluentbit
	helm upgrade --install fluent-bit vendor-charts/fluentbit \
		--namespace $(MONITORING_NAMESPACE) --create-namespace

install-grafana:
	@echo "🔧 Instalando Grafana..."
	helm dependency update vendor-charts/grafana
	helm upgrade --install grafana vendor-charts/grafana \
		--namespace $(MONITORING_NAMESPACE) --create-namespace

install-external-secrets:
	@echo "🔧 Instalando External Secrets Operator..."
	helm dependency update vendor-charts/external-secrets
	helm upgrade --install external-secrets vendor-charts/external-secrets \
		--namespace $(EXTERNAL_SECRETS_NAMESPACE) --create-namespace

install-monitoring-stack: install-prometheus install-loki install-fluentbit install-grafana
	@echo "✅ Stack de monitorización instalado"

update-dependencies:
	@echo "🔄 Actualizando dependencias de charts..."
	helm dependency update vendor-charts/juicefs-csi-driver
	helm dependency update vendor-charts/argo-cd
	helm dependency update vendor-charts/prometheus-stack
	helm dependency update vendor-charts/loki
	helm dependency update vendor-charts/fluentbit
	helm dependency update vendor-charts/grafana
	helm dependency update vendor-charts/external-secrets

update-dependencies-all: update-dependencies
	@echo "✅ Todas las dependencias actualizadas"

# Help
help:
	@echo "make create-namespace          # para crear el namespace"
	@echo "make deploy                    # para desplegar la aplicación"
	@echo "make status                    # para ver el estado"
	@echo "make logs                      # para ver los logs"
	@echo "make delete                    # para eliminar la aplicación"
	@echo "make install-juicefs           # para instalar JuiceFS CSI Driver"
	@echo "make install-argocd            # para instalar Argo CD"
	@echo "make install-prometheus        # para instalar Prometheus Stack"
	@echo "make install-loki              # para instalar Loki"
	@echo "make install-fluentbit         # para instalar Fluent Bit"
	@echo "make install-grafana           # para instalar Grafana"
	@echo "make install-external-secrets  # para instalar External Secrets Operator"
	@echo "make install-monitoring-stack  # para instalar todo el stack de monitorización"
	@echo "make update-dependencies       # para actualizar dependencias de charts"
	@echo "make update-dependencies-all   # para actualizar todas las dependencias"
