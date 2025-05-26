output "opensearch_dashboards_url" {
  description = "OpenSearch Dashboards service endpoint."
  value       = "http://localhost:5601 (via port-forwarding to svc/${helm_release.opensearch_dashboards.name}.${var.namespace}.svc.cluster.local:5601)"
}

output "opensearch_dashboards_service_name" {
    description = "Name of the OpenSearch Dashboards service."
    value = helm_release.opensearch_dashboards.name
} 