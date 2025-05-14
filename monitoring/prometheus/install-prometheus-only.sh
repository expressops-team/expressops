#!/bin/bash


helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

if helm list -n monitoring | grep -q "prometheus-custom"; then
  helm uninstall prometheus-custom -n monitoring
  sleep 3
fi

kubectl create namespace monitoring 2>/dev/null || true
helm install prometheus-d prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values prometheus-only-values.yaml

sleep 10

kubectl get pods -n monitoring | grep prometheus-custom
