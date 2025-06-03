#!/bin/bash


kubectl apply -f elasticsearch-deployment.yaml
kubectl -n dav-monitoring wait --for=condition=ready pod -l app=elasticsearch --timeout=300s
kubectl get pods -n dav-monitoring -l app=elasticsearch
kubectl get pvc -n dav-monitoring elasticsearch-data -o wide
echo "curl http://localhost:9200/_cat/indices?v"