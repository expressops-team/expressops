#!/bin/bash
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${BLUE}🔄 Añadiendo repositorio de Prometheus...${NC}"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

echo -e "${BLUE}🔄 Verificando instalaciones previas de Prometheus...${NC}"
if helm list -n monitoring | grep -q "prometheus-custom"; then
  echo -e "${YELLOW}⚠️ Instalación previa encontrada. Desinstalando...${NC}"
  helm uninstall prometheus-custom -n monitoring
  echo -e "${GREEN}✅ Instalación previa eliminada${NC}"
  sleep 3
fi

echo -e "${BLUE}🔄 Verificando namespace monitoring...${NC}"
kubectl create namespace monitoring 2>/dev/null || true
echo -e "${BLUE}🔄 Instalando Prometheus personalizado...${NC}"
helm install prometheus-d prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values prometheus-only-values.yaml

echo -e "${GREEN}✅ Prometheus instalado correctamente${NC}"
echo -e "${YELLOW}⏳ Esperando a que el pod esté listo (puede tardar un minuto)...${NC}"
sleep 10

echo -e "${BLUE}🔄 Verificando estado de la instalación...${NC}"
kubectl get pods -n monitoring | grep prometheus-custom

echo -e "\n${BLUE}===========================================================${NC}"
echo -e "${GREEN}✅ INSTALACIÓN COMPLETADA${NC}"
echo -e "${BLUE}===========================================================${NC}"
echo -e "${YELLOW}Para acceder a Prometheus UI mediante port-forward:${NC}"
echo -e "kubectl port-forward -n monitoring svc/prometheus-custom-kube-prometheus-prometheus 9090:9090"
echo -e "${BLUE}===========================================================${NC}" 