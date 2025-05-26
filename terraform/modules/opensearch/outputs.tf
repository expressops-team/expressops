output "opensearch_url" {
  description = "OpenSearch service endpoint."
  value       = "https://${helm_release.opensearch.name}-cluster-master.${var.namespace}.svc.cluster.local:9200"
}

output "opensearch_service_name" {
    description = "Name of the OpenSearch cluster master service."
    value = "${helm_release.opensearch.name}-cluster-master" 
} 