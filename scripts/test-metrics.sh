#!/bin/bash

# Script to test metrics updates in ExpressOps
# Usage: ./scripts/test-metrics.sh

HOST=${1:-"localhost:8080"}
INTERVAL=${2:-5}

echo "Testing metrics updates at http://$HOST/metrics"
echo "Interval: $INTERVAL seconds between checks"
echo "============================================="

get_metric_value() {
  local metric_name=$1
  curl -s "http://$HOST/metrics" | grep "^$metric_name" | grep -v "#" | awk '{print $2}'
}

print_metric() {
  local name=$1
  local value=$(get_metric_value "$name")
  if [ -n "$value" ]; then
    echo "$name: $value"
  else
    echo "$name: Not found"
  fi
}

while true; do
  echo "Timestamp: $(date)"
  
  print_metric "expressops_cpu_usage_percent"
  print_metric "expressops_memory_usage_bytes"
  print_metric "expressops_storage_usage_bytes"
  print_metric "expressops_concurrent_plugins"
  
  echo -n "Executing health-check: "
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://$HOST/flow?flowName=health-status-no-format")
  if [ "$HTTP_CODE" == "200" ]; then
    echo "OK ($HTTP_CODE)"
  else
    echo "Error ($HTTP_CODE)"
  fi
  
  echo "============================================="
  
  sleep $INTERVAL
done