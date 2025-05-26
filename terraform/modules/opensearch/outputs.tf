output "opensearch_url" {
  description = "OpenSearch service endpoint."
  value       = "https://${helm_release.opensearch.name}-cluster-master.${var.namespace}.svc.cluster.local:9200"
}

output "opensearch_service_name" {
    value = "${helm_release.opensearch.name}-cluster-master" # Default from chart
}

output "debug_opensearch_values_file_exists" {
  value       = fileexists(var.opensearch_values_file)
} 