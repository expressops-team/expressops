#!/bin/bash

# Usage: ./generate-traffic.sh [URL] [REQUESTS] [DELAY_SECONDS]

URL=${1:-"http://localhost:8081/flow"}
REQUESTS=${2:-100}
DELAY=${3:-0.5}

echo "Generating $REQUESTS requests to $URL with a $DELAY second delay between requests"

FLOWS=(
  "all-flows"
  "create-user"
  "set-permissions"
  "full-user-onboarding"
  "health-status-no-format"
  "alert-health"
  "debug-test"
  "kube-notify"
  "full-monitoring"
)

for i in $(seq 1 $REQUESTS); do
  FLOW=${FLOWS[$RANDOM % ${#FLOWS[@]}]}
  
  echo "Request $i - Ejecutando flujo: $FLOW"
  
  curl -s "$URL?flowName=$FLOW" > /dev/null
  
  sleep $DELAY
done

echo "Tráfico generado. Verifica tus métricas en Prometheus/Grafana." 