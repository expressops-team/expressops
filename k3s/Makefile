APP_NAME=expressops
NAMESPACE=expressops-dev
YAML_DEPLOYMENT=expressops.yaml
YAML_CONFIGMAP=configmap.yaml

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

# Help
help:
	@echo "make create-namespace  # para crear el namespace"
	@echo "make deploy            # para desplegar la aplicación"
	@echo "make status            # para ver el estado"
	@echo "make logs              # para ver los logs"
	@echo "make delete            # para eliminar la aplicación"
