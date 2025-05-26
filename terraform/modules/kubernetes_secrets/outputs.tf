output "opensearch_certs_secret_name" {
  description = "OpenSearch TLS certs secret name"
  value       = kubernetes_secret.opensearch_certs.metadata[0].name
}

output "opensearch_credentials_secret_name" {
  description = "OpenSearch admin credentials secret name"
  value       = kubernetes_secret.opensearch_credentials.metadata[0].name
} 