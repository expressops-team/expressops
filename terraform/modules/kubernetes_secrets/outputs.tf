output "opensearch_certs_secret_name" {
  value       = kubernetes_secret.opensearch_certs.metadata[0].name
}

output "opensearch_credentials_secret_name" {
  value       = kubernetes_secret.opensearch_credentials.metadata[0].name
} 