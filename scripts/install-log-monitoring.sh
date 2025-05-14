#!/bin/bash

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}Starting log monitoring stack installation in namespace dav-monitoring${NC}"

kubectl create namespace dav-monitoring 2>/dev/null || echo -e "${GREEN}Namespace dav-monitoring already exists${NC}"

echo -e "${YELLOW}Adding Helm repositories...${NC}"
helm repo add opensearch https://opensearch-project.github.io/helm-charts/
helm repo add elastic https://helm.elastic.co
helm repo update

export OPENSEARCH_INITIAL_ADMIN_PASSWORD=${OPENSEARCH_INITIAL_ADMIN_PASSWORD:-"admin123"}
echo -e "${YELLOW}Using admin password for OpenSearch: ${OPENSEARCH_INITIAL_ADMIN_PASSWORD}${NC}"
echo -e "${YELLOW}To change the password, run: export OPENSEARCH_INITIAL_ADMIN_PASSWORD=your_password${NC}"

echo -e "${YELLOW}Installing OpenSearch...${NC}"
helm upgrade --install opensearch-cluster opensearch/opensearch \
  --namespace dav-monitoring \
  --values monitoring/opensearch/opensearch-values.yaml \
  --set securityConfig.adminCredentials.password="${OPENSEARCH_INITIAL_ADMIN_PASSWORD}"

echo -e "${YELLOW}Waiting for OpenSearch to be ready...${NC}"
kubectl rollout status statefulset/opensearch-cluster-master -n dav-monitoring --timeout=300s

echo -e "${YELLOW}Installing OpenSearch Dashboards...${NC}"
helm upgrade --install opensearch-dashboards opensearch/opensearch-dashboards \
  --namespace dav-monitoring \
  --values monitoring/opensearch/opensearch-dashboards-values.yaml \
  --set opensearch.username=admin \
  --set opensearch.password="${OPENSEARCH_INITIAL_ADMIN_PASSWORD}"

echo -e "${YELLOW}Installing Logstash...${NC}"
helm upgrade --install logstash elastic/logstash \
  --namespace dav-monitoring \
  --values monitoring/logstash/logstash-values.yaml \
  --set extraEnvs[0].value=admin \
  --set extraEnvs[1].value="${OPENSEARCH_INITIAL_ADMIN_PASSWORD}"

echo -e "${YELLOW}Installing Filebeat...${NC}"
helm upgrade --install filebeat elastic/filebeat \
  --namespace dav-monitoring \
  --values monitoring/filebeat/filebeat-values.yaml

echo -e "${GREEN}Installation completed!${NC}"
echo -e "${YELLOW}To access OpenSearch Dashboards:${NC}"
echo -e "kubectl port-forward svc/opensearch-dashboards 5601:5601 -n dav-monitoring"
echo -e "${YELLOW}Then visit: http://localhost:5601${NC}"
echo -e "${YELLOW}Credentials: admin / ${OPENSEARCH_INITIAL_ADMIN_PASSWORD}${NC}"