#!/bin/bash

echo "Configuring ExpressOps logging chain "

echo "Applying RBAC permissions for Filebeat..."
kubectl apply -f monitoring/filebeat/filebeat-rbac.yaml

echo "Applying Filebeat ConfigMap with Logstash output..."
kubectl apply -f monitoring/filebeat/filebeat-alt-configmap.yaml

echo "Applying Filebeat DaemonSet..."
kubectl apply -f monitoring/filebeat/filebeat-daemonset.yaml

echo "Deploying Logstash..."
kubectl apply -f monitoring/logstash/logstash-configmap.yaml
kubectl apply -f monitoring/logstash/logstash-service.yaml
kubectl apply -f monitoring/logstash/logstash-deployment.yaml

echo "Waiting for pods to be ready..."
kubectl rollout status daemonset/filebeat -n dav-monitoring
kubectl rollout status deployment/logstash -n dav-monitoring

echo  "Verifying pod status "
kubectl get pods -n dav-monitoring -l app=filebeat
kubectl get pods -n dav-monitoring -l app=logstash

echo  "Configuration completed "
echo "To view Filebeat logs: kubectl logs -n dav-monitoring -l app=filebeat"
echo "To view Logstash logs: kubectl logs -n dav-monitoring -l app=logstash" # if u update is deprecated ;D
echo "Once logs are generated, check OpenSearch Dashboard index: expressops-logs-*"

chmod +x monitoring/deploy-logging.sh