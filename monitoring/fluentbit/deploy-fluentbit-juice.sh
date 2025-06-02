#!/bin/bash

kubectl create namespace dav-monitoring --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f fluentbit-rbac-juice.yaml

kubectl apply -f fluentbit-configmap-juice.yaml

kubectl apply -f fluentbit-daemonset-juice.yaml

kubectl -n dav-monitoring wait --for=condition=ready pod -l app.kubernetes.io/name=fluentbit-dual --timeout=120s

kubectl get pods -n dav-monitoring -l app.kubernetes.io/name=fluentbit-dual

kubectl get svc -n dav-monitoring -l app.kubernetes.io/name=fluentbit-dual

kubectl exec -n dav-monitoring -l app.kubernetes.io/name=fluentbit-dual -- df -h /fluent-bit/tail_db