#!/bin/bash

OPENSEARCH_HOST="localhost"
OPENSEARCH_PORT="9201"
USERNAME="admin"
PASSWORD="admin"

curl -k -X GET "https://$OPENSEARCH_HOST:$OPENSEARCH_PORT/_cat/plugins?v" -u $USERNAME:$PASSWORD | grep ml

curl -k -X GET "https://$OPENSEARCH_HOST:$OPENSEARCH_PORT/system-metrics/_count" -u $USERNAME:$PASSWORD

KMEANS_RESPONSE=$(curl -k -X POST "https://$OPENSEARCH_HOST:$OPENSEARCH_PORT/_plugins/_ml/models/_train" -u $USERNAME:$PASSWORD \
-H "Content-Type: application/json" \
-d '{
  "name": "system_metrics_kmeans",
  "algorithm": "kmeans",
  "parameters": {
    "centroids": 3,
    "iterations": 10,
    "distance_type": "EUCLIDEAN"
  },
  "input_query": {
    "query": {"match_all": {}},
    "size": 1000
  },
  "input_index": ["system-metrics"],
  "include_fields": ["cpu_usage", "memory_usage"]
}')

echo "$KMEANS_RESPONSE"

KMEANS_RESPONSE2=$(curl -k -X POST "https://$OPENSEARCH_HOST:$OPENSEARCH_PORT/_plugins/_ml/train/kmeans" -u $USERNAME:$PASSWORD \
-H "Content-Type: application/json" \
-d '{
  "name": "system_metrics_clusters",
  "parameters": {
    "centroids": 3,
    "iterations": 10,
    "distance_type": "EUCLIDEAN"
  },
  "input_data": {
    "indices": ["system-metrics"],
    "query": {"match_all": {}},
    "includes": ["cpu_usage", "memory_usage"]
  }
}')

echo "$KMEANS_RESPONSE2"

curl -k -X GET "https://$OPENSEARCH_HOST:$OPENSEARCH_PORT/_plugins/_ml/models/_search?pretty" -u $USERNAME:$PASSWORD \
-H "Content-Type: application/json" \
-d '{
  "query": {
    "match_all": {}
  }
}'